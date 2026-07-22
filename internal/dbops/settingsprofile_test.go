package dbops

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

const (
	testProfileID   = "0e4a8a40-dd60-b0d9-7baa-d44f89302d8d"
	testProfileName = "test-14xyz-use1_readwrite_default"
)

// httpAlreadyExistsErr mimics the opaque error returned by the HTTP client:
// the raw ClickHouse response body wrapped in message-only context.
var httpAlreadyExistsErr = errors.WithMessage(
	errors.WithMessage(
		errors.New("Code: 493. DB::Exception: settings profile `test-14xyz-use1_readwrite_default`: cannot insert because settings profile `test-14xyz-use1_readwrite_default` already exists in `replicated`. (ACCESS_ENTITY_ALREADY_EXISTS) (version 26.3.3.20 (official build))"),
		"error executing query",
	),
	"error running query",
)

// nativeAlreadyExistsErr mimics the error returned by the native client: a
// typed *clickhouse.Exception wrapped in message-only context. The Exception
// message deliberately lacks the "Code: 493." HTTP body format so this
// exercises the typed detection path.
var nativeAlreadyExistsErr = errors.WithMessage(
	&clickhouse.Exception{
		Code:    493,
		Name:    "ACCESS_ENTITY_ALREADY_EXISTS",
		Message: "settings profile `test-14xyz-use1_readwrite_default`: cannot insert because settings profile `test-14xyz-use1_readwrite_default` already exists in `replicated`",
	},
	"error executing query",
)

// fakeClickhouseClient is a scripted clickhouseclient.ClickhouseClient.
//
// Exec returns execErr. Select routes on the query shape used by the settings
// profile dbops functions; the first findMisses find-by-name lookups return
// zero rows to simulate replication/visibility lag.
type fakeClickhouseClient struct {
	execErr    error
	findMisses int

	execCalls int
	findCalls int
}

func (f *fakeClickhouseClient) Exec(_ context.Context, _ string, _ ...map[string]string) error {
	f.execCalls++
	return f.execErr
}

func (f *fakeClickhouseClient) Select(_ context.Context, qry string, callback func(clickhouseclient.Row) error) error {
	switch {
	case strings.Contains(qry, "settings_profile_elements"):
		// Inherited profiles lookup: none.
		return nil
	case strings.Contains(qry, "toString("):
		// FindSettingsProfileByName: SELECT toString(id) ... WHERE name = ...
		f.findCalls++
		if f.findCalls <= f.findMisses {
			return nil
		}
		row := clickhouseclient.Row{}
		row.Set("id", testProfileID)
		return callback(row)
	case strings.Contains(qry, "settings_profiles"):
		// GetSettingsProfile: SELECT name ... WHERE id = ...
		row := clickhouseclient.Row{}
		row.Set("name", testProfileName)
		return callback(row)
	default:
		return fmt.Errorf("unexpected query: %s", qry)
	}
}

func newTestClient(t *testing.T, fake *fakeClickhouseClient) Client {
	t.Helper()

	client, err := NewClient(fake, WithReadAfterWriteTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	return client
}

// TestCreateSettingsProfile_AdoptsExisting verifies the recovery path for
// ClickHouse error code 493 (ACCESS_ENTITY_ALREADY_EXISTS): when the profile
// already exists (a previous apply created it but its ID was never recorded
// in state), Create adopts it by name instead of failing.
func TestCreateSettingsProfile_AdoptsExisting(t *testing.T) {
	tests := []struct {
		name    string
		execErr error
	}{
		{name: "HTTP protocol error shape", execErr: httpAlreadyExistsErr},
		{name: "Native protocol error shape", execErr: nativeAlreadyExistsErr},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := &fakeClickhouseClient{execErr: tc.execErr}
			client := newTestClient(t, fake)

			profile, err := client.CreateSettingsProfile(context.Background(), SettingsProfile{Name: testProfileName}, nil)
			if err != nil {
				t.Fatalf("expected adoption of existing settings profile, got error: %v", err)
			}
			if profile == nil {
				t.Fatal("expected a settings profile, got nil")
			}
			if profile.ID != testProfileID {
				t.Errorf("expected adopted profile ID %q, got %q", testProfileID, profile.ID)
			}
			if profile.Name != testProfileName {
				t.Errorf("expected adopted profile name %q, got %q", testProfileName, profile.Name)
			}
		})
	}
}

// TestCreateSettingsProfile_AdoptsExistingAfterLookupMiss covers the full
// create -> miss -> retry -> reconcile flow: the create hits 493, the first
// lookups by name return no rows (visibility lag), and the adoption succeeds
// once the profile becomes visible, without aborting the backoff window.
func TestCreateSettingsProfile_AdoptsExistingAfterLookupMiss(t *testing.T) {
	fake := &fakeClickhouseClient{execErr: httpAlreadyExistsErr, findMisses: 2}
	client := newTestClient(t, fake)

	profile, err := client.CreateSettingsProfile(context.Background(), SettingsProfile{Name: testProfileName}, nil)
	if err != nil {
		t.Fatalf("expected adoption after transient lookup misses, got error: %v", err)
	}
	if profile == nil {
		t.Fatal("expected a settings profile, got nil")
	}
	if profile.ID != testProfileID {
		t.Errorf("expected adopted profile ID %q, got %q", testProfileID, profile.ID)
	}
	if fake.findCalls < 3 {
		t.Errorf("expected at least 3 find-by-name attempts (2 misses + 1 hit), got %d", fake.findCalls)
	}
}

// TestCreateSettingsProfile_UnrelatedErrorStillFails verifies that only
// already-exists errors trigger adoption; any other create failure is
// returned as-is without attempting a lookup.
func TestCreateSettingsProfile_UnrelatedErrorStillFails(t *testing.T) {
	fake := &fakeClickhouseClient{
		execErr: errors.New("Code: 241. DB::Exception: Memory limit (total) exceeded (MEMORY_LIMIT_EXCEEDED)"),
	}
	client := newTestClient(t, fake)

	profile, err := client.CreateSettingsProfile(context.Background(), SettingsProfile{Name: testProfileName}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if profile != nil {
		t.Errorf("expected nil profile, got %+v", profile)
	}
	if fake.findCalls != 0 {
		t.Errorf("expected no find-by-name attempts on unrelated error, got %d", fake.findCalls)
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

// TestFindSettingsProfileByName_NotFound verifies the not-found contract used
// by retryWithBackoff: a lookup miss returns (nil, nil) rather than an error,
// so callers keep retrying within their backoff window instead of aborting.
func TestFindSettingsProfileByName_NotFound(t *testing.T) {
	fake := &fakeClickhouseClient{findMisses: 1}
	client := newTestClient(t, fake)

	profile, err := client.FindSettingsProfileByName(context.Background(), testProfileName, nil)
	if err != nil {
		t.Fatalf("expected (nil, nil) on lookup miss, got error: %v", err)
	}
	if profile != nil {
		t.Errorf("expected nil profile on lookup miss, got %+v", profile)
	}
}
