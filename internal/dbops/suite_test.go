package dbops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/stretchr/testify/require"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/chcontainer"
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

// countingClient counts SELECTs containing `match` and fails the first `failures` of them.
type countingClient struct {
	clickhouseclient.ClickhouseClient
	match    string
	selects  atomic.Int32
	failures atomic.Int32
}

func (c *countingClient) Select(ctx context.Context, sql string, callback func(clickhouseclient.Row) error) error {
	if strings.Contains(sql, c.match) {
		c.selects.Add(1)
		if c.failures.Add(-1) >= 0 {
			return errors.New("injected failure")
		}
	}
	return c.ClickhouseClient.Select(ctx, sql, callback)
}

// execHookClient calls onExec before running every statement.
type execHookClient struct {
	clickhouseclient.ClickhouseClient
	onExec func()
}

func (c *execHookClient) Exec(ctx context.Context, sql string, params ...map[string]string) error {
	if c.onExec != nil {
		c.onExec()
	}
	return c.ClickhouseClient.Exec(ctx, sql, params...)
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

func TestExecParams(t *testing.T) {
	server := chcontainer.Start(t, "")
	values := map[string]string{
		"backslash pair":  `x\\y`,
		"backslash n":     `x\ny`,
		"tab":             "x\ty",
		"newline":         "x\ny",
		"carriage return": "x\ry",
		"quote":           `x'y`,
		"placeholder":     "{other:String}",
	}

	for _, protocol := range protocols {
		raw := server.Client(t, protocol)
		client := newTestClient(t, raw)
		for name, value := range values {
			t.Run(protocol+"/"+name, func(t *testing.T) {
				table := acctest.RandomWithPrefix("tfacc")
				execOnCleanup(t, raw, fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table))
				require.NoError(t, raw.Exec(context.Background(), fmt.Sprintf("CREATE TABLE `%s` ENGINE = Memory AS SELECT {v:String} AS v", table), map[string]string{"v": value}))
				var got string
				require.NoError(t, raw.Select(context.Background(), fmt.Sprintf("SELECT v FROM `%s`", table), func(row clickhouseclient.Row) error {
					var err error
					got, err = row.GetString("v")
					return err
				}))
				require.Equal(t, value, got)

				user := acctest.RandomWithPrefix("tfacc")
				execOnCleanup(t, raw, fmt.Sprintf("DROP USER IF EXISTS `%s`", user))
				_, err := client.CreateUser(context.Background(), User{Name: user, AuthMethods: []AuthMethod{{Type: "sha256_password", Args: []string{value}}}}, nil)
				require.NoError(t, err)
				as := server
				as.Username, as.Password = user, value
				require.NoError(t, as.Client(t, protocol).Exec(context.Background(), "SELECT 1"))
			})
		}
	}
}
