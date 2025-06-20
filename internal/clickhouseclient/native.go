package clickhouseclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"reflect"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pingcap/errors"
)

const defaultDatabase = "default"

type nativeClient struct {
	connection driver.Conn
}

type NativeClientConfig struct {
	Host             string
	Port             uint16
	UserPasswordAuth *UserPasswordAuth
	EnableTLS        bool
}

func NewNativeClient(config NativeClientConfig) (ClickhouseClient, error) {
	if config.Host == "" {
		return nil, errors.New("Host is required")
	}
	if config.Port == 0 {
		return nil, errors.New("Port is required")
	}
	if config.UserPasswordAuth == nil {
		return nil, errors.New("Exactly one authentication method is required")
	}

	options := clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
	}

	if config.UserPasswordAuth != nil {
		auth := clickhouse.Auth{}
		auth.Database = config.UserPasswordAuth.Database
		auth.Username = config.UserPasswordAuth.Username
		auth.Password = config.UserPasswordAuth.Password

		if auth.Database == "" {
			auth.Database = defaultDatabase
		}

		options.Auth = auth
	}

	if config.EnableTLS {
		options.TLS = &tls.Config{} //nolint:gosec
	}

	conn, err := clickhouse.Open(&options)
	if err != nil {
		return nil, err
	}

	// Default timeout of native client is 30 seconds.
	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelFunc()

	err = conn.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return &nativeClient{
		connection: conn,
	}, nil
}

func (i *nativeClient) Select(ctx context.Context, qry string, callback func(Row) error) error {
	ctx = tflog.SetField(ctx, "Query", qry)
	tflog.Debug(ctx, "Running Query")

	rows, err := i.connection.Query(ctx, qry)
	if err != nil {
		return errors.WithMessage(err, "error executing query")
	}

	// Prepare a slice of variable pointers dynamically typed based on the query result's column types.
	columnTypes := rows.ColumnTypes()
	vars := make([]any, len(columnTypes))
	for i := range columnTypes {
		vars[i] = reflect.New(columnTypes[i].ScanType()).Interface()
	}

	// Scan each row of the result.
	for i := 0; rows.Next(); i++ {
		// Read the columns using the dynamically created variables.
		if err := rows.Scan(vars...); err != nil {
			return errors.WithMessage(err, "error scanning row")
		}

		// Prepare a Row for the callback.
		ret := Row{}
		for i, v := range vars {
			switch v := v.(type) {
			case *string:
				// Non-nullable string, return string value.
				ret.Set(rows.Columns()[i], *v)
			case *uuid.UUID:
				// Return string representation.
				ret.Set(rows.Columns()[i], v.String())
			case **string:
				// Nullable string, return either nil or a pointer to the string
				ret.Set(rows.Columns()[i], *v)
			case *uint8:
				ret.Set(rows.Columns()[i], *v)
			case *uint64:
				ret.Set(rows.Columns()[i], *v)
			default:
				return errors.New(fmt.Sprintf("unsupported column type: %s", reflect.TypeOf(v)))
			}
		}
		err = callback(ret)
		if err != nil {
			return errors.WithMessage(err, "error populating Row from query result")
		}
	}

	return nil
}

func (i *nativeClient) Exec(ctx context.Context, qry string) error {
	ctx = tflog.SetField(ctx, "Query", qry)
	tflog.Debug(ctx, "Running Query")

	err := i.connection.Exec(ctx, qry)
	if err != nil {
		return errors.WithMessage(err, "error executing query")
	}

	return nil
}
