package dbops

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// retryWithBackoff retries a retrieval function that may return nil due to lag.
// It uses exponential backoff with context-aware cancellation.
//
// The retrieval function should:
//   - Return (result, nil) when the resource is found
//   - Return (nil, nil) when the resource is not found (will trigger retry)
//   - Return (nil, error) when an actual error occurs (will abort retry)
//
// Parameters:
//   - ctx: Context for cancellation
//   - resourceType: Human-readable resource type name (e.g., "user", "role")
//   - resourceIdentifier: The specific identifier being looked up (e.g., user name)
//   - retrievalFunc: Function that attempts to retrieve the resource
//   - timeout: Optional timeout duration. If not provided, defaults to 30 seconds.
//
// Returns the retrieved resource or an error if all retries are exhausted.
func retryWithBackoff[T any](
	ctx context.Context,
	resourceType string,
	resourceIdentifier string,
	retrievalFunc func() (*T, error),
	timeout ...time.Duration,
) (*T, error) {
	const defaultTimeout = 30 * time.Second
	const initialBackoff = 50 * time.Millisecond

	// Use provided timeout or default
	retryTimeout := defaultTimeout
	if len(timeout) > 0 {
		retryTimeout = timeout[0]
	}

	// Create context with timeout
	retryCtx, cancel := context.WithTimeout(ctx, retryTimeout)
	defer cancel()

	backoff := initialBackoff
	for {
		result, err := retrievalFunc()
		if err != nil {
			return nil, fmt.Errorf("error retrieving created %s: %w", resourceType, err)
		}

		if result != nil {
			return result, nil
		}

		tflog.Debug(ctx, fmt.Sprintf("%s not found, retrying with exponential backoff", resourceType), map[string]any{
			"resource_identifier": resourceIdentifier,
			"backoff":             backoff.String(),
		})

		// Context-aware sleep with exponential backoff
		timer := time.NewTimer(backoff)
		select {
		case <-retryCtx.Done():
			timer.Stop()
			return nil, fmt.Errorf(
				"%s %q was created but could not be retrieved within timeout (%v): %w",
				resourceType, resourceIdentifier, retryTimeout, retryCtx.Err(),
			)
		case <-timer.C:
			backoff *= 2
		}
	}
}
