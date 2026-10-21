package dbops

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

var protocols = []string{"native", "http"}

// laggingClient simulates read-after-write lag: the first `misses` SELECTs containing `match` return no rows.
type laggingClient struct {
	clickhouseclient.ClickhouseClient
	match  string
	misses int
}

func (c *laggingClient) Select(ctx context.Context, sql string, callback func(clickhouseclient.Row) error) error {
	if c.misses > 0 && strings.Contains(sql, c.match) {
		c.misses--
		return nil
	}
	return c.ClickhouseClient.Select(ctx, sql, callback)
}

func newTestClient(t *testing.T, chClient clickhouseclient.ClickhouseClient) Client {
	t.Helper()

	client, err := NewClient(chClient, WithReadAfterWriteTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	return client
}

func mustExec(t *testing.T, raw clickhouseclient.ClickhouseClient, sql string) {
	t.Helper()

	if err := raw.Exec(context.Background(), sql); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

func execOnCleanup(t *testing.T, raw clickhouseclient.ClickhouseClient, sql string) {
	t.Cleanup(func() {
		if err := raw.Exec(context.Background(), sql); err != nil {
			t.Errorf("%s: %v", sql, err)
		}
	})
}
