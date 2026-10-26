package dbops

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

type impl struct {
	clickhouseClient      clickhouseclient.ClickhouseClient
	CapabilityFlags       *CapabilityFlags
	readAfterWriteTimeout time.Duration
	initOnce              sync.Once
	initErr               error
	grantsCache           sync.Map
	grantsFills           singleflight.Group
	replicatedStorage     atomic.Pointer[bool]
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
		CapabilityFlags: nil,
	}
	c.clickhouseClient = &cacheClearingClient{client: clickhouseClient, cache: &c.grantsCache}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// cacheClearingClient clears the grants cache after every Exec, as any of them may change access entities.
type cacheClearingClient struct {
	client clickhouseclient.ClickhouseClient
	cache  *sync.Map
}

func (c *cacheClearingClient) Select(ctx context.Context, qry string, callback func(clickhouseclient.Row) error) error {
	return c.client.Select(ctx, qry, callback)
}

func (c *cacheClearingClient) Exec(ctx context.Context, qry string, params ...map[string]string) error {
	defer c.cache.Clear()
	return c.client.Exec(ctx, qry, params...)
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
		i.CapabilityFlags, i.initErr = i.detectCapabilityFlags(ctx)
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
