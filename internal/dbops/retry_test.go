package dbops

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestRetryWithBackoff_ImmediateSuccess tests that retry returns immediately when resource is found
func TestRetryWithBackoff_ImmediateSuccess(t *testing.T) {
	ctx := context.Background()
	expectedResult := &testResource{ID: "test-123", Name: "test"}
	callCount := 0

	result, err := retryWithBackoff(
		ctx,
		"test resource",
		"test-123",
		func() (*testResource, error) {
			callCount++
			return expectedResult, nil
		},
		1*time.Second,
	)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != expectedResult {
		t.Errorf("Expected result %v, got %v", expectedResult, result)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}
}

// TestRetryWithBackoff_EventualSuccess tests retry succeeds after a few attempts
func TestRetryWithBackoff_EventualSuccess(t *testing.T) {
	ctx := context.Background()
	expectedResult := &testResource{ID: "test-456", Name: "delayed"}
	callCount := 0

	result, err := retryWithBackoff(
		ctx,
		"test resource",
		"test-456",
		func() (*testResource, error) {
			callCount++
			// Return nil for first 3 calls to simulate lag
			if callCount < 3 {
				return nil, nil
			}
			return expectedResult, nil
		},
		5*time.Second,
	)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if result != expectedResult {
		t.Errorf("Expected result %v, got %v", expectedResult, result)
	}
	if callCount < 3 {
		t.Errorf("Expected at least 3 calls, got %d", callCount)
	}
}

// TestRetryWithBackoff_ImmediateError tests that actual errors abort retry
func TestRetryWithBackoff_ImmediateError(t *testing.T) {
	ctx := context.Background()
	expectedError := errors.New("database connection failed")
	callCount := 0

	result, err := retryWithBackoff(
		ctx,
		"test resource",
		"test-789",
		func() (*testResource, error) {
			callCount++
			return nil, expectedError
		},
		1*time.Second,
	)

	if err == nil {
		t.Error("Expected an error, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 call (no retry on error), got %d", callCount)
	}
	if !errors.Is(err, expectedError) {
		t.Errorf("Expected error to wrap original error")
	}
}

// TestRetryWithBackoff_Timeout tests that retry times out when resource never appears
func TestRetryWithBackoff_Timeout(t *testing.T) {
	ctx := context.Background()
	callCount := 0

	start := time.Now()
	result, err := retryWithBackoff(
		ctx,
		"test resource",
		"never-found",
		func() (*testResource, error) {
			callCount++
			return nil, nil // Always return not found
		},
		200*time.Millisecond, // Short timeout for faster test
	)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
	if callCount < 2 {
		t.Errorf("Expected multiple retry attempts, got %d", callCount)
	}
	// Verify timeout was respected (with some margin)
	if elapsed < 200*time.Millisecond || elapsed > 500*time.Millisecond {
		t.Errorf("Expected elapsed time ~200ms, got %v", elapsed)
	}
}

// testResource is a helper struct for testing
type testResource struct {
	ID   string
	Name string
}
