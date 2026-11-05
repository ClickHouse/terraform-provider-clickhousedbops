package clickhouseclient

import (
	"context"
	"strings"
)

type ClickhouseClient interface {
	Select(ctx context.Context, qry string, callback func(Row) error) error
	// Exec runs a query, only the first params entry passed to the server.
	Exec(ctx context.Context, qry string, params ...map[string]string) error
}

// The server decodes query parameter values as TSV-escaped text over both HTTP and native.
var paramEscaper = strings.NewReplacer(`\`, `\\`, "\t", `\t`, "\n", `\n`)

func escapeParams(params map[string]string) map[string]string {
	escaped := make(map[string]string, len(params))
	for k, v := range params {
		escaped[k] = paramEscaper.Replace(v)
	}
	return escaped
}
