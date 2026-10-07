package dbops

import (
	"context"
	"strings"
	"testing"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

func TestSplitSystemAccessObject(t *testing.T) {
	tests := []struct {
		name       string
		accessType string
		value      string
		wantObject string
		wantFilter *string
	}{
		{
			name:       "quoted source filter",
			accessType: "READ",
			value:      "URL(`https://example\\\\.com/files/.*`)",
			wantObject: "URL",
			wantFilter: new(`https://example\.com/files/.*`),
		},
		{
			name:       "unquoted source filter",
			accessType: "WRITE",
			value:      "S3(bucket_name)",
			wantObject: "S3",
			wantFilter: new("bucket_name"),
		},
		{
			name:       "unfiltered source",
			accessType: "READ",
			value:      "URL",
			wantObject: "URL",
		},
		{
			name:       "non-source access object with parentheses stays opaque",
			accessType: "CREATE USER",
			value:      "team(foo)",
			wantObject: "team(foo)",
		},
		{
			name:       "unknown source name stays opaque",
			accessType: "READ",
			value:      "CUSTOM(foo)",
			wantObject: "CUSTOM(foo)",
		},
		{
			name:       "malformed source filter stays opaque",
			accessType: "READ",
			value:      "URL(`unterminated)",
			wantObject: "URL(`unterminated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			object, filter := splitSystemAccessObject(tt.accessType, new(tt.value))
			if object == nil || *object != tt.wantObject {
				t.Fatalf("object = %v, want %q", object, tt.wantObject)
			}
			if !equalStringPointers(filter, tt.wantFilter) {
				t.Fatalf("filter = %v, want %v", filter, tt.wantFilter)
			}
		})
	}
}

func TestAccessObjectMatchesDistinguishesSourceFilters(t *testing.T) {
	filter := `https://example\.com/files/.*`

	if !accessObjectMatches("READ", new("URL"), &filter, new("URL(`https://example\\\\.com/files/.*`)")) {
		t.Fatal("expected semantically identical filtered source targets to match")
	}
	if accessObjectMatches("READ", new("URL"), &filter, new("URL")) {
		t.Fatal("filtered source target must not match an unfiltered source grant")
	}
	if accessObjectMatches("READ", new("URL"), nil, new("URL(`https://example\\\\.com/files/.*`)")) {
		t.Fatal("unfiltered source target must not match a filtered source grant")
	}
}

func TestGetAllGrantsPreservesSourceFilterIdentity(t *testing.T) {
	role := "reader"
	client := &queuedClickHouseClient{
		selectRows: [][]clickhouseclient.Row{{
			grantRow("READ", "URL(`https://example\\\\.com/files/.*`)", role),
		}},
	}
	dbopsClient, err := NewClient(client)
	if err != nil {
		t.Fatal(err)
	}

	grants, err := dbopsClient.GetAllGrantsForGrantee(context.Background(), nil, &role, nil)
	if err != nil {
		t.Fatalf("GetAllGrantsForGrantee() error = %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("grant count = %d, want 1", len(grants))
	}
	if grants[0].AccessObject == nil || *grants[0].AccessObject != "URL" {
		t.Fatalf("access object = %v, want URL", grants[0].AccessObject)
	}
	wantFilter := `https://example\.com/files/.*`
	if !equalStringPointers(grants[0].AccessObjectFilter, &wantFilter) {
		t.Fatalf("filter = %v, want %q", grants[0].AccessObjectFilter, wantFilter)
	}
}

func TestGrantPrivilegeRollsBackBroadenedSourceFilter(t *testing.T) {
	role := "reader"
	filter := `https://example\.com/files/.*`
	broadSourceRow := grantRow("READ", "URL", role)

	client := &queuedClickHouseClient{
		selectRows: [][]clickhouseclient.Row{
			{versionRow()},
			nil,              // No unfiltered grant before the GRANT.
			{broadSourceRow}, // Filtered lookup must reject the broad row.
			{broadSourceRow}, // Broad lookup detects unsafe server behavior.
		},
	}
	dbopsClient, err := NewClient(client)
	if err != nil {
		t.Fatal(err)
	}

	_, err = dbopsClient.GrantPrivilege(context.Background(), GrantPrivilege{
		AccessType:          "READ",
		AccessObject:        new("URL"),
		AccessObjectFilter:  &filter,
		GranteeRoleName:     &role,
		ParameterizedTarget: true,
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "rolled back the broader grant") {
		t.Fatalf("GrantPrivilege() error = %v, want rollback error", err)
	}

	if len(client.execQueries) != 2 {
		t.Fatalf("Exec query count = %d, want grant and rollback", len(client.execQueries))
	}
	if !strings.Contains(client.execQueries[0], "GRANT READ ON `URL`(") {
		t.Fatalf("first query = %q, want filtered grant", client.execQueries[0])
	}
	if client.execQueries[1] != "REVOKE READ ON `URL` FROM `reader`;" {
		t.Fatalf("rollback query = %q", client.execQueries[1])
	}
}

func TestGrantPrivilegeFindsExactFilterAmongMultipleSourceGrants(t *testing.T) {
	role := "reader"
	filter := `https://example\.com/files/.*`

	client := &queuedClickHouseClient{
		selectRows: [][]clickhouseclient.Row{
			{versionRow()},
			nil, // No unfiltered grant before the GRANT.
			{
				grantRow("READ", "URL(`https://example\\\\.com/files/.*`)", role),
				grantRow("READ", "URL(`https://other\\\\.example/.*`)", role),
			},
		},
	}
	dbopsClient, err := NewClient(client)
	if err != nil {
		t.Fatal(err)
	}

	grant, err := dbopsClient.GrantPrivilege(context.Background(), GrantPrivilege{
		AccessType:          "READ",
		AccessObject:        new("URL"),
		AccessObjectFilter:  &filter,
		GranteeRoleName:     &role,
		ParameterizedTarget: true,
	}, nil)
	if err != nil {
		t.Fatalf("GrantPrivilege() error = %v", err)
	}
	if grant == nil || !equalStringPointers(grant.AccessObjectFilter, &filter) {
		t.Fatalf("GrantPrivilege() = %#v, want filtered grant", grant)
	}
	if len(client.execQueries) != 1 {
		t.Fatalf("Exec query count = %d, want only the grant", len(client.execQueries))
	}
}

type queuedClickHouseClient struct {
	selectRows  [][]clickhouseclient.Row
	execQueries []string
}

func (c *queuedClickHouseClient) Select(_ context.Context, _ string, callback func(clickhouseclient.Row) error) error {
	rows := c.selectRows[0]
	c.selectRows = c.selectRows[1:]
	for _, row := range rows {
		if err := callback(row); err != nil {
			return err
		}
	}
	return nil
}

func (c *queuedClickHouseClient) Exec(_ context.Context, query string, _ ...map[string]string) error {
	c.execQueries = append(c.execQueries, query)
	return nil
}

func versionRow() clickhouseclient.Row {
	var row clickhouseclient.Row
	row.Set("value", "26.4.1.1")
	return row
}

func grantRow(accessType, accessObject, role string) clickhouseclient.Row {
	var row clickhouseclient.Row
	row.Set("access_type", accessType)
	row.Set("database", (*string)(nil))
	row.Set("table", (*string)(nil))
	row.Set("column", (*string)(nil))
	row.Set("access_object_nullable", new(accessObject))
	row.Set("user_name", (*string)(nil))
	row.Set("role_name", new(role))
	row.Set("grant_option", false)
	return row
}
