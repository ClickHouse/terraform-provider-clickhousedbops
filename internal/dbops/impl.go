package dbops

import (
	"context"
	"sync"
	"time"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

type impl struct {
	clickhouseClient      clickhouseclient.ClickhouseClient
	CapabilityFlags       *CapabilityFlags
	readAfterWriteTimeout time.Duration
	initOnce              sync.Once
	initErr               error
}

// ClientOption configures optional behaviour of the dbops client.
type ClientOption func(*impl)

// WithReadAfterWriteTimeout sets the timeout used for read-after-write
// verification of created resources.
func WithReadAfterWriteTimeout(d time.Duration) ClientOption {
	return func(i *impl) {
		i.readAfterWriteTimeout = d
	}
}

func NewClient(clickhouseClient clickhouseclient.ClickhouseClient, opts ...ClientOption) (Client, error) {
	c := &impl{
		clickhouseClient: clickhouseClient,
		CapabilityFlags:  nil,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// readAfterWriteTimeoutArgs returns the configured timeout as a variadic-compatible
// slice. Returns nil when unconfigured, causing retryWithBackoff to use its default.
func (i *impl) readAfterWriteTimeoutArgs() []time.Duration {
	if i.readAfterWriteTimeout > 0 {
		return []time.Duration{i.readAfterWriteTimeout}
	}
	return nil
}

func (i *impl) initCapabilities(ctx context.Context) {
	i.initOnce.Do(func() {
		version, err := i.GetVersion(ctx)
		if err != nil {
			i.initErr = err
			return
		}
		i.CapabilityFlags = NewCapabilityFlags(version)
	})
}

// Returns initialized capability flags for the connected ClickHouse server.
func (i *impl) GetCapabilityFlags(ctx context.Context) (CapabilityFlags, error) {
	i.initCapabilities(ctx)

	if i.CapabilityFlags == nil {
		return CapabilityFlags{}, i.initErr
	}

	return *i.CapabilityFlags, i.initErr
}
