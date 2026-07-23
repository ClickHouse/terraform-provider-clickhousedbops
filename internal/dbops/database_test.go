package dbops

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

type databaseClickhouseClient struct {
	selectQueries   []string
	selectRows      []clickhouseclient.Row
	execQueries     []string
	sensitiveQuery  string
	redactedQuery   string
	sensitiveValues []string
}

func (c *databaseClickhouseClient) Select(_ context.Context, query string, callback func(clickhouseclient.Row) error) error {
	c.selectQueries = append(c.selectQueries, query)
	rows := c.selectRows
	if len(rows) == 0 {
		rows = []clickhouseclient.Row{databaseRow("00000000-0000-0000-0000-000000000001", "test database", "DataLakeCatalog")}
	}
	for _, row := range rows {
		if err := callback(row); err != nil {
			return err
		}
	}
	return nil
}

func (c *databaseClickhouseClient) Exec(_ context.Context, query string, _ ...map[string]string) error {
	c.execQueries = append(c.execQueries, query)
	return nil
}

func (c *databaseClickhouseClient) ExecSensitive(_ context.Context, query string, redactedQuery string, sensitiveValues []string) error {
	c.sensitiveQuery = query
	c.redactedQuery = redactedQuery
	c.sensitiveValues = sensitiveValues
	return nil
}

func databaseRow(uuid, comment, engine string) clickhouseclient.Row {
	return namedDatabaseRow(uuid, "catalog", comment, engine)
}

func namedDatabaseRow(uuid, name, comment, engine string) clickhouseclient.Row {
	row := clickhouseclient.Row{}
	row.Set("uuid", uuid)
	row.Set("name", name)
	row.Set("comment", comment)
	row.Set("engine", engine)
	return row
}

func TestGetDatabaseRejectsUUIDSharedByMultipleNames(t *testing.T) {
	const zeroUUID = "00000000-0000-0000-0000-000000000000"
	clickhouse := &databaseClickhouseClient{
		selectRows: []clickhouseclient.Row{
			namedDatabaseRow(zeroUUID, "information_schema", "", "Memory"),
			namedDatabaseRow(zeroUUID, "catalog", "", "Memory"),
		},
	}
	client := &impl{clickhouseClient: clickhouse}

	_, err := client.GetDatabase(context.Background(), zeroUUID, nil)
	if err == nil || !strings.Contains(err.Error(), "matches multiple database names") || !strings.Contains(err.Error(), "by name") {
		t.Fatalf("GetDatabase() error = %v, want ambiguous UUID error recommending name identity", err)
	}
}

func TestFindDatabaseByNameUsesOneLogicalLookup(t *testing.T) {
	clickhouse := &databaseClickhouseClient{
		selectRows: []clickhouseclient.Row{
			databaseRow("00000000-0000-0000-0000-000000000003", "test database", "DataLakeCatalog"),
			databaseRow("00000000-0000-0000-0000-000000000001", "test database", "DataLakeCatalog"),
			databaseRow("00000000-0000-0000-0000-000000000002", "test database", "DataLakeCatalog"),
		},
	}
	client := &impl{clickhouseClient: clickhouse}

	database, err := client.FindDatabaseByName(context.Background(), "catalog", new("cluster"))
	if err != nil {
		t.Fatalf("FindDatabaseByName() error = %v", err)
	}
	if database == nil || database.Name != "catalog" || database.Engine != "DataLakeCatalog" {
		t.Fatalf("FindDatabaseByName() = %#v", database)
	}
	if database.UUID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("FindDatabaseByName() UUID = %q, want deterministic minimum", database.UUID)
	}
	if len(clickhouse.selectQueries) != 1 {
		t.Fatalf("FindDatabaseByName() issued %d queries, want 1", len(clickhouse.selectQueries))
	}
	for _, field := range []string{"uuid", "name", "comment", "engine"} {
		if !strings.Contains(clickhouse.selectQueries[0], "`"+field+"`") {
			t.Errorf("query %q does not select %q", clickhouse.selectQueries[0], field)
		}
	}
}

func TestFindDatabaseByNameRejectsInconsistentReplicaMetadata(t *testing.T) {
	clickhouse := &databaseClickhouseClient{
		selectRows: []clickhouseclient.Row{
			databaseRow("00000000-0000-0000-0000-000000000001", "first", "Atomic"),
			databaseRow("00000000-0000-0000-0000-000000000002", "second", "Memory"),
		},
	}
	client := &impl{clickhouseClient: clickhouse}

	_, err := client.FindDatabaseByName(context.Background(), "catalog", new("cluster"))
	if err == nil || !strings.Contains(err.Error(), "inconsistent metadata") {
		t.Fatalf("FindDatabaseByName() error = %v, want inconsistent metadata error", err)
	}
}

func TestCreateDatabaseRedactsWriteOnlyEngineParameters(t *testing.T) {
	clickhouse := &databaseClickhouseClient{}
	client := &impl{clickhouseClient: clickhouse}

	_, err := client.CreateDatabase(context.Background(), Database{
		Name:            "catalog",
		Engine:          "DataLakeCatalog",
		EngineArguments: []string{"'https://catalog.example.test'"},
		EngineSettings: map[string]string{ //nolint:gosec // Contains a placeholder, not a credential.
			"catalog_type":       "'rest'",
			"catalog_credential": "{credential:String}",
		},
		EngineParameters: map[string]string{"credential": "top-secret"},
	}, nil)
	if err != nil {
		t.Fatalf("CreateDatabase() error = %v", err)
	}
	if len(clickhouse.execQueries) != 0 {
		t.Fatalf("CreateDatabase() used ordinary Exec for a sensitive query")
	}
	if !strings.Contains(clickhouse.sensitiveQuery, "'top-secret'") {
		t.Errorf("executed query does not contain safely quoted parameter: %q", clickhouse.sensitiveQuery)
	}
	if strings.Contains(clickhouse.redactedQuery, "top-secret") || !strings.Contains(clickhouse.redactedQuery, "'[REDACTED]'") {
		t.Errorf("redacted query exposes secret or lacks marker: %q", clickhouse.redactedQuery)
	}
	if !slices.Equal(clickhouse.sensitiveValues, []string{"top-secret"}) {
		t.Errorf("sensitive values = %q, want [top-secret]", clickhouse.sensitiveValues)
	}
}
