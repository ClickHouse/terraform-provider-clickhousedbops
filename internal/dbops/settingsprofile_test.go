package dbops

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

const (
	testProfileID   = "0e4a8a40-dd60-b0d9-7baa-d44f89302d8d"
	testProfileName = "test-14xyz-use1_readwrite_default"
)

const (
	createProfileSQL = "CREATE SETTINGS PROFILE `test-14xyz-use1_readwrite_default`;"
	findByNameSQL    = "SELECT toString(`id`) AS `id` FROM `system`.`settings_profiles` WHERE (`name` = 'test-14xyz-use1_readwrite_default');"
	getByIDSQL       = "SELECT `name` FROM `system`.`settings_profiles` WHERE (`id` = '0e4a8a40-dd60-b0d9-7baa-d44f89302d8d');"
	inheritSQL       = "SELECT `inherit_profile` FROM `system`.`settings_profile_elements` WHERE (`profile_name` = 'test-14xyz-use1_readwrite_default') ORDER BY `index` ASC;"
)

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

type step struct {
	wantSQL string
	rows    map[string]string
	err     error
}

// scriptedClient serves queries from an ordered script and fails the test on any deviation.
type scriptedClient struct {
	t     *testing.T
	steps []step
}

func (c *scriptedClient) next(sql string) step {
	c.t.Helper()
	if len(c.steps) == 0 {
		c.t.Fatalf("unexpected query: %s", sql)
	}
	s := c.steps[0]
	c.steps = c.steps[1:]
	if sql != s.wantSQL {
		c.t.Fatalf("query mismatch:\nwant: %s\ngot:  %s", s.wantSQL, sql)
	}
	return s
}

func (c *scriptedClient) Exec(_ context.Context, sql string, _ ...map[string]string) error {
	return c.next(sql).err
}

func (c *scriptedClient) Select(_ context.Context, sql string, callback func(clickhouseclient.Row) error) error {
	s := c.next(sql)
	if s.err != nil {
		return s.err
	}
	for field, value := range s.rows {
		row := clickhouseclient.Row{}
		row.Set(field, value)
		if err := callback(row); err != nil {
			return err
		}
	}
	return nil
}

func newTestClient(t *testing.T, script []step) (Client, *scriptedClient) {
	t.Helper()

	fake := &scriptedClient{t: t, steps: script}
	client, err := NewClient(fake, WithReadAfterWriteTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	return client, fake
}

func TestCreateSettingsProfile(t *testing.T) {
	miss := step{wantSQL: findByNameSQL}
	found := []step{
		{wantSQL: findByNameSQL, rows: map[string]string{"id": testProfileID}},
		{wantSQL: getByIDSQL, rows: map[string]string{"name": testProfileName}},
		{wantSQL: inheritSQL},
	}

	tests := []struct {
		name    string
		script  []step
		wantErr bool
	}{
		{
			name:   "adopts existing profile after transient lookup misses",
			script: append([]step{{wantSQL: createProfileSQL, err: httpAlreadyExistsErr}, miss, miss}, found...),
		},
		{
			name:   "retries transient lookup misses after successful create",
			script: append([]step{{wantSQL: createProfileSQL}, miss, miss}, found...),
		},
		{
			name:    "fails on unrelated error without lookup",
			script:  []step{{wantSQL: createProfileSQL, err: errors.New("Code: 241. DB::Exception: Memory limit (total) exceeded (MEMORY_LIMIT_EXCEEDED)")}},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, fake := newTestClient(t, tc.script)

			profile, err := client.CreateSettingsProfile(context.Background(), SettingsProfile{Name: testProfileName}, nil)

			if len(fake.steps) != 0 {
				t.Errorf("script not fully consumed, %d steps left", len(fake.steps))
			}

			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if profile != nil {
					t.Errorf("expected nil profile, got %+v", profile)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected settings profile, got error: %v", err)
			}
			if profile == nil {
				t.Fatal("expected a settings profile, got nil")
			}
			if profile.ID != testProfileID {
				t.Errorf("expected profile ID %q, got %q", testProfileID, profile.ID)
			}
			if profile.Name != testProfileName {
				t.Errorf("expected profile name %q, got %q", testProfileName, profile.Name)
			}
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
	client, _ := newTestClient(t, []step{{wantSQL: findByNameSQL}})

	profile, err := client.FindSettingsProfileByName(context.Background(), testProfileName, nil)
	if err != nil {
		t.Fatalf("expected (nil, nil) on lookup miss, got error: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile on lookup miss, got %+v", profile)
	}
}
