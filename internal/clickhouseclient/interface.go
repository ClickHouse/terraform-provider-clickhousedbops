package clickhouseclient

import (
	"context"
)

type ClickhouseClient interface {
	Select(ctx context.Context, qry string, callback func(Row) error) error
	// Exec runs a query, only the first params entry passed to the server.
	Exec(ctx context.Context, qry string, params ...map[string]string) error
}
