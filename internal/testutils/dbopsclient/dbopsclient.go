package dbopsclient

import (
	"context"
	"fmt"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

const (
	username = "default"
	password = "test"

	host       = "127.0.0.1"
	nativePort = 9000
	httpPort   = 8123
)

type ConnectionSettings struct {
	Host     string
	Port     uint16
	Username string
	Password string
}

func NewDbopsClient(protocol string) (dbopsClient dbops.Client, connectionSettings ConnectionSettings, err error) {
	connectionSettings = ConnectionSettings{
		Host:     host,
		Username: username,
		Password: password,
	}

	var clickhouseClient clickhouseclient.ClickhouseClient
	{
		switch protocol {
		case "native":
			connectionSettings.Port = nativePort
			config := clickhouseclient.NativeClientConfig{
				Host: host,
				Port: nativePort,
				UserPasswordAuth: &clickhouseclient.UserPasswordAuth{
					Username: username,
					Password: password,
				},
			}

			clickhouseClient, err = clickhouseclient.NewNativeClient(config)
			if err != nil {
				return
			}
		case "http":
			connectionSettings.Port = httpPort
			config := clickhouseclient.HTTPClientConfig{
				Protocol: "http",
				Host:     host,
				Port:     httpPort,
				BasicAuth: &clickhouseclient.BasicAuth{
					Username: username,
					Password: password,
				},
			}

			clickhouseClient, err = clickhouseclient.NewHTTPClient(config)
			if err != nil {
				return
			}
		default:
			err = fmt.Errorf("invalid protocol %s", protocol)
			return
		}
	}

	dbopsClient, err = dbops.NewClient(clickhouseClient)
	if err != nil {
		return
	}

	return
}

// SecondReplicaID returns the id of the access entity `name` in `table` (e.g. system.roles) on the second replica of
// cluster1, which the tests don't connect to. With localfile storage every replica gives the entity its own id, so it
// fails when the first replica has the same id.
func SecondReplicaID(ctx context.Context, table string, name string) (string, error) {
	clickhouseClient, err := clickhouseclient.NewNativeClient(clickhouseclient.NativeClientConfig{
		Host: host,
		Port: nativePort,
		UserPasswordAuth: &clickhouseclient.UserPasswordAuth{
			Username: username,
			Password: password,
		},
	})
	if err != nil {
		return "", err
	}

	ids := make([]string, 0, 2)
	for _, from := range []string{table, fmt.Sprintf("remote('tests-clickhouse-2', %s, '%s', '%s')", table, username, password)} {
		var id string
		sql := fmt.Sprintf("SELECT toString(id) AS id FROM %s WHERE %s", from, querybuilder.WhereEquals("name", name).Clause())
		err = clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
			id, err = data.GetString("id")
			return err
		})
		if err != nil {
			return "", err
		}
		if id == "" {
			return "", fmt.Errorf("%q was not found in %s", name, from)
		}
		ids = append(ids, id)
	}

	if ids[0] == ids[1] {
		return "", fmt.Errorf("both replicas give %q the id %q, but with localfile storage they should differ", name, ids[0])
	}

	return ids[1], nil
}
