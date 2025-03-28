package clickhouseclient

import (
	"context"
)

type ClickhouseClient interface {
	Select(ctx context.Context, qry string, callback func(Row) error) error
	Exec(ctx context.Context, qry string) error
}
