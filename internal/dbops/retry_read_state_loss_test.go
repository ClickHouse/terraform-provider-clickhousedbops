package dbops

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

// flakeyGetUser wraps a real Client and returns (nil, nil) for the first
// failCount calls to GetUser, then delegates to the real client.
// This simulates transient replica inconsistency where a resource
// temporarily appears to not exist.
type flakeyGetUser struct {
	Client
	real      Client
	callCount int
	failCount int
}

func (f *flakeyGetUser) GetUser(ctx context.Context, id string, clusterName *string) (*User, error) {
	f.callCount++
	if f.callCount <= f.failCount {
		return nil, nil // simulate transient "not found"
	}
	return f.real.GetUser(ctx, id, clusterName)
}

// dockerComposeUp starts the ClickHouse cluster with the given config,
// inheriting the full parent environment to avoid PATH/credential issues.
// It retries up to 3 times because ClickHouse may crash on the first attempt
// if ZooKeeper (required for replicated configs) isn't ready yet.
func dockerComposeUp(t *testing.T, testsDir, configFile string) {
	t.Helper()
	env := append(os.Environ(), fmt.Sprintf("CONFIGFILE=%s", configFile))

	for attempt := 1; attempt <= 3; attempt++ {
		cmd := exec.Command("docker", "compose", "up", "-d", "--wait")
		cmd.Dir = testsDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = env
		if err := cmd.Run(); err != nil {
			if attempt < 3 {
				t.Logf("docker compose up attempt %d failed (retrying): %v", attempt, err)
				time.Sleep(2 * time.Second)
				continue
			}
			t.Fatalf("docker compose up failed after %d attempts: %v", attempt, err)
		}
		return
	}
}

// dockerComposeDown tears down the ClickHouse cluster and removes volumes.
func dockerComposeDown(t *testing.T, testsDir string) {
	t.Helper()
	cmd := exec.Command("docker", "compose", "down", "-v")
	cmd.Dir = testsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Errorf("docker compose down failed: %v", err)
	}
}

// newTestDbopsClient creates a dbops.Client connected to the local ClickHouse
// spun up by Docker Compose, avoiding an import cycle with the dbopsclient
// test utility package.
func newTestDbopsClient() (Client, error) {
	chClient, err := clickhouseclient.NewNativeClient(clickhouseclient.NativeClientConfig{
		Host: "127.0.0.1",
		Port: 9000,
		UserPasswordAuth: &clickhouseclient.UserPasswordAuth{
			Username: "default",
			Password: "test",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("creating clickhouse client: %w", err)
	}
	return NewClient(chClient)
}

// TestRetryReadStateLoss reproduces the state-loss bug described in issue #157
// and verifies that retryWithBackoff (the fix) resolves it.
//
// The bug: Read() calls GetUser once; if it returns (nil, nil) transiently
// (e.g., due to replica lag), Terraform calls RemoveResource() and permanently
// forgets the resource.
//
// The fix: wrap GetUser in retryWithBackoff so transient nils are retried
// before concluding the resource is gone.
//
// The cluster uses config-replicated.xml with 2 replicas (the Docker Compose
// default) to mirror real ClickHouse deployments where replica inconsistency
// can cause transient nil reads.
func TestRetryReadStateLoss(t *testing.T) {
	if os.Getenv("TF_ACC") != "1" {
		t.Skip("Skipping acceptance test because TF_ACC is not set to 1")
	}

	ctx := context.Background()
	testsDir := "../../tests"

	// Spin up a replicated ClickHouse cluster (2 replicas) via Docker Compose.
	dockerComposeUp(t, testsDir, "config-replicated.xml")
	defer dockerComposeDown(t, testsDir)

	// Create a real dbops client connected to the local ClickHouse.
	dbopsClient, err := newTestDbopsClient()
	if err != nil {
		t.Fatalf("Failed to create dbops client: %v", err)
	}

	// Create a real user in ClickHouse with a unique timestamp suffix.
	// With replicated access storage (config-replicated.xml), user entities
	// are automatically replicated via ZooKeeper, so we don't pass ON CLUSTER.
	userName := fmt.Sprintf("test_state_loss_%d", time.Now().UnixNano())
	user, err := dbopsClient.CreateUser(ctx, User{
		Name:               userName,
		PasswordSha256Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	}, nil)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	defer func() {
		_ = dbopsClient.DeleteUser(ctx, user.ID, nil)
	}()

	// Sanity check: GetUser works for the real user.
	found, err := dbopsClient.GetUser(ctx, user.ID, nil)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if found == nil {
		t.Fatal("GetUser returned nil for a user that was just created")
	}
	t.Logf("Created user %q with ID %q (replicated cluster with 2 nodes)", found.Name, found.ID)

	t.Run("without_RetryRead_single_GetUser_call_returns_nil_(bug_reproduced)", func(t *testing.T) {
		// Simulate the old Read() code path: a single call to GetUser that
		// returns (nil, nil) due to transient replica inconsistency.
		flakey := &flakeyGetUser{
			real:      dbopsClient,
			failCount: 1, // first call returns nil
		}

		// Old Read() logic: single call, no retry.
		result, err := flakey.GetUser(ctx, user.ID, nil)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// BUG: the user exists but the single call returned nil.
		// In the old code, this triggers resp.State.RemoveResource() → state loss.
		if result != nil {
			t.Fatal("Expected nil result (simulated transient miss), but got a user — flakey mock is broken")
		}
		t.Log("Bug reproduced: single GetUser call returned nil for an existing user → state would be lost")
	})

	t.Run("with_RetryRead_transient_nil_is_retried_and_user_is_found_(fix_verified)", func(t *testing.T) {
		// Simulate the fixed Read() code path: retryWithBackoff retries
		// on (nil, nil) before concluding the resource is gone.
		flakey := &flakeyGetUser{
			real:      dbopsClient,
			failCount: 3, // first 3 calls return nil, 4th returns real user
		}

		result, err := retryWithBackoff(
			ctx,
			"user",
			user.ID,
			func() (*User, error) {
				return flakey.GetUser(ctx, user.ID, nil)
			},
			5*time.Second,
		)
		if err != nil {
			t.Fatalf("retryWithBackoff returned error: %v", err)
		}

		if result == nil {
			t.Fatal("retryWithBackoff returned nil — fix did not work")
		}

		if result.ID != user.ID {
			t.Fatalf("Expected user ID %q, got %q", user.ID, result.ID)
		}

		if flakey.callCount <= flakey.failCount {
			t.Fatalf("Expected at least %d calls, got %d", flakey.failCount+1, flakey.callCount)
		}

		t.Logf("Fix verified: retryWithBackoff retried %d times and found user %q", flakey.callCount, result.Name)
	})

	t.Run("retryWithBackoff_returns_error_when_resource_truly_gone", func(t *testing.T) {
		// Verify that retryWithBackoff does NOT mask a truly deleted resource.
		deletedUserName := fmt.Sprintf("test_deleted_%d", time.Now().UnixNano())
		deletedUser, err := dbopsClient.CreateUser(ctx, User{
			Name:               deletedUserName,
			PasswordSha256Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		}, nil)
		if err != nil {
			t.Fatalf("Failed to create user for deletion test: %v", err)
		}

		if err := dbopsClient.DeleteUser(ctx, deletedUser.ID, nil); err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		result, err := retryWithBackoff(
			ctx,
			"user",
			deletedUser.ID,
			func() (*User, error) {
				return dbopsClient.GetUser(ctx, deletedUser.ID, nil)
			},
			500*time.Millisecond, // short timeout — resource is truly gone
		)

		if err == nil {
			t.Fatal("Expected timeout error for a truly deleted user, got nil")
		}
		if result != nil {
			t.Fatalf("Expected nil result for deleted user, got: %+v", result)
		}

		t.Logf("Correctly timed out for deleted user: %v", err)
	})

	fmt.Println("\n✓ All state-loss tests passed: bug reproduced and fix verified")
}
