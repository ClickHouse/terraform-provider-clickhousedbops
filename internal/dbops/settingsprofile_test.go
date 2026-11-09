package dbops

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/pingcap/errors"
	"github.com/stretchr/testify/require"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/chcontainer"
)

const findProfileByNameSQL = "`system`.`settings_profiles` WHERE (`name` = "

var httpAlreadyExistsErr = errors.WithMessage(
	errors.WithMessage(
		errors.New("Code: 493. DB::Exception: settings profile `test-14xyz-use1_readwrite_default`: cannot insert because settings profile `test-14xyz-use1_readwrite_default` already exists in `replicated`. (ACCESS_ENTITY_ALREADY_EXISTS) (version 26.3.3.20 (official build))"),
		"error executing query",
	),
	"error running query",
)

var nativeAlreadyExistsErr = errors.WithMessage(
	&clickhouse.Exception{
		Code:    493,
		Name:    "ACCESS_ENTITY_ALREADY_EXISTS",
		Message: "settings profile `test-14xyz-use1_readwrite_default`: cannot insert because settings profile `test-14xyz-use1_readwrite_default` already exists in `replicated`",
	},
	"error executing query",
)

func TestCreateSettingsProfile(t *testing.T) {
	server := chcontainer.Start(t, "")
	missingProfile := acctest.RandomWithPrefix("tfacc")

	tests := []struct {
		name           string
		existing       bool
		inheritFrom    []string
		lookupMisses   int
		wantErr        bool
		wantMissesLeft int
	}{
		{name: "adopts existing profile after transient lookup misses", existing: true, lookupMisses: 2},
		{name: "retries transient lookup misses after successful create", lookupMisses: 2},
		{name: "fails on unrelated error without lookup", inheritFrom: []string{missingProfile}, lookupMisses: 2, wantErr: true, wantMissesLeft: 2},
	}

	for _, protocol := range protocols {
		raw := server.Client(t, protocol)
		lagging := &laggingClient{ClickhouseClient: raw, match: findProfileByNameSQL}
		client := newTestClient(t, lagging)
		direct := newTestClient(t, raw)

		for _, tc := range tests {
			t.Run(protocol+"/"+tc.name, func(t *testing.T) {
				lagging.misses = tc.lookupMisses
				name := acctest.RandomWithPrefix("tfacc")
				execOnCleanup(t, raw, fmt.Sprintf("DROP SETTINGS PROFILE IF EXISTS `%s`", name))
				var existingID string
				if tc.existing {
					mustExec(t, raw, fmt.Sprintf("CREATE SETTINGS PROFILE `%s`", name))
					existing, err := retryWithBackoff(context.Background(), "settings profile", name, func() (*SettingsProfile, error) {
						return direct.FindSettingsProfileByName(context.Background(), name, nil)
					})
					require.NoError(t, err)
					require.NotNil(t, existing)
					existingID = existing.ID
				}

				created, err := client.CreateSettingsProfile(context.Background(), SettingsProfile{Name: name, InheritFrom: tc.inheritFrom}, nil)

				if tc.wantErr {
					require.Error(t, err)
					require.Nil(t, created)
				} else {
					require.NoError(t, err)
					require.NotNil(t, created)
					require.Equal(t, name, created.Name)
					require.NotEmpty(t, created.ID)
					if tc.existing {
						require.Equal(t, existingID, created.ID, "adopted profile ID")
					}
				}
				require.Equal(t, tc.wantMissesLeft, lagging.misses, "lookup misses left")
			})
		}
	}
}

func TestUpdateSettingsProfile(t *testing.T) {
	server := chcontainer.Start(t, "")

	for _, protocol := range protocols {
		raw := server.Client(t, protocol)
		client := newTestClient(t, raw)

		t.Run(protocol, func(t *testing.T) {
			ctx := context.Background()
			parent := acctest.RandomWithPrefix("tfacc")
			name := acctest.RandomWithPrefix("tfacc")
			renamed := acctest.RandomWithPrefix("tfacc")
			execOnCleanup(t, raw, fmt.Sprintf("DROP SETTINGS PROFILE IF EXISTS `%s`, `%s`, `%s`", name, renamed, parent))
			mustExec(t, raw, fmt.Sprintf("CREATE SETTINGS PROFILE `%s`", parent))

			created, err := client.CreateSettingsProfile(ctx, SettingsProfile{Name: name, InheritFrom: []string{parent}}, nil)
			require.NoError(t, err)
			_, err = client.CreateSetting(ctx, created.ID, Setting{Name: "max_threads", Value: new("4")}, nil, 5*time.Second)
			require.NoError(t, err)

			updated, err := client.UpdateSettingsProfile(ctx, SettingsProfile{ID: created.ID, Name: renamed, InheritFrom: []string{"default", parent}}, nil)
			require.NoError(t, err)
			require.Equal(t, renamed, updated.Name)
			require.Equal(t, []string{"default", parent}, updated.InheritFrom)

			setting, err := client.GetSetting(ctx, created.ID, "max_threads", nil)
			require.NoError(t, err)
			require.NotNil(t, setting, "settings must survive a profile update")
		})
	}
}

func TestIsAlreadyExistsError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "unrelated error", err: errors.New("connection refused"), want: false},
		{name: "unrelated ClickHouse error code", err: errors.New("Code: 241. DB::Exception: Memory limit (total) exceeded"), want: false},
		{name: "HTTP body with code 493", err: httpAlreadyExistsErr, want: true},
		{name: "typed native exception", err: nativeAlreadyExistsErr, want: true},
		{
			name: "typed native exception behind stdlib wrapper",
			err:  fmt.Errorf("outer: %w", &clickhouse.Exception{Code: 493, Name: "ACCESS_ENTITY_ALREADY_EXISTS"}),
			want: true,
		},
		{
			name: "typed native exception with non-493 code",
			err:  errors.WithMessage(&clickhouse.Exception{Code: 241, Name: "MEMORY_LIMIT_EXCEEDED", Message: "oom"}, "error executing query"),
			want: false,
		},
		{name: "already exists name without code", err: errors.New("whatever (ACCESS_ENTITY_ALREADY_EXISTS)"), want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAlreadyExistsError(tc.err); got != tc.want {
				t.Errorf("isAlreadyExistsError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestFindSettingsProfileByName_NotFound verifies the not-found contract used by retryWithBackoff:
// a lookup miss returns (nil, nil) rather than an error, so callers keep retrying within their backoff.
func TestFindSettingsProfileByName_NotFound(t *testing.T) {
	server := chcontainer.Start(t, "")

	for _, protocol := range protocols {
		client := newTestClient(t, server.Client(t, protocol))

		t.Run(protocol, func(t *testing.T) {
			profile, err := client.FindSettingsProfileByName(context.Background(), acctest.RandomWithPrefix("tfacc"), nil)
			require.NoError(t, err, "expected (nil, nil) on lookup miss")
			require.Nil(t, profile, "expected nil profile on lookup miss")
		})
	}
}
