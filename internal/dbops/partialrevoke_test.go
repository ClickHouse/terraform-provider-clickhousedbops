package dbops

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
)

type partialRevokeTestClient struct {
	execQueries   []string
	selectQueries []string
	selectFunc    func(string, func(clickhouseclient.Row) error) error
}

func (c *partialRevokeTestClient) Exec(_ context.Context, query string, _ ...map[string]string) error {
	c.execQueries = append(c.execQueries, query)
	return nil
}

func (c *partialRevokeTestClient) Select(_ context.Context, query string, callback func(clickhouseclient.Row) error) error {
	c.selectQueries = append(c.selectQueries, query)
	if c.selectFunc == nil {
		return nil
	}
	return c.selectFunc(query, callback)
}

func accessTypeRow(accessType string) clickhouseclient.Row {
	var row clickhouseclient.Row
	row.Set("access_type", accessType)
	return row
}

func partialRevokeRow(accessType string, database, table, column, accessObject, user, role *string, grantOptionOnly bool) clickhouseclient.Row {
	var row clickhouseclient.Row
	row.Set("access_type", accessType)
	row.Set("database", database)
	row.Set("table", table)
	row.Set("column", column)
	row.Set("access_object_nullable", accessObject)
	row.Set("user_name", user)
	row.Set("role_name", role)
	row.Set("grant_option", grantOptionOnly)
	return row
}

func positiveGrantRow(accessType string, database, table, column, accessObject, user, role *string, grantOption bool) clickhouseclient.Row {
	return partialRevokeRow(accessType, database, table, column, accessObject, user, role, grantOption)
}

func TestCreatePartialRevoke(t *testing.T) {
	client := &partialRevokeTestClient{
		selectFunc: func(query string, callback func(clickhouseclient.Row) error) error {
			if strings.Contains(query, "`is_partial_revoke` = 1") {
				return callback(accessTypeRow("SELECT"))
			}
			return nil
		},
	}
	dbClient := &impl{clickhouseClient: client, readAfterWriteTimeout: time.Second}
	partial := PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		TableName:       new("events"),
		ColumnName:      new("secret"),
		GranteeRoleName: new("analyst"),
	}

	got, err := dbClient.CreatePartialRevoke(context.Background(), partial, new("cluster"))
	if err != nil {
		t.Fatalf("CreatePartialRevoke() error = %v", err)
	}
	if got == nil {
		t.Fatal("CreatePartialRevoke() returned nil")
	}
	if len(client.execQueries) != 1 {
		t.Fatalf("Exec query count = %d, want 1", len(client.execQueries))
	}
	want := "REVOKE ON CLUSTER 'cluster' SELECT(`secret`) ON `analytics`.`events` FROM `analyst`;"
	if client.execQueries[0] != want {
		t.Errorf("Exec query = %q, want %q", client.execQueries[0], want)
	}
}

func TestCreatePartialRevoke_GrantOptionOnly(t *testing.T) {
	client := &partialRevokeTestClient{
		selectFunc: func(_ string, callback func(clickhouseclient.Row) error) error {
			return callback(accessTypeRow("SELECT"))
		},
	}
	dbClient := &impl{clickhouseClient: client, readAfterWriteTimeout: time.Second}
	partial := PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		GranteeUserName: new("reporter"),
		GrantOptionOnly: true,
	}

	if _, err := dbClient.CreatePartialRevoke(context.Background(), partial, nil); err != nil {
		t.Fatalf("CreatePartialRevoke() error = %v", err)
	}
	want := "REVOKE GRANT OPTION FOR SELECT ON `analytics`.* FROM `reporter`;"
	if client.execQueries[0] != want {
		t.Errorf("Exec query = %q, want %q", client.execQueries[0], want)
	}
}

func TestCreatePartialRevoke_ReturnsNilWhenBroaderNegativeRightCoversIt(t *testing.T) {
	selectCount := 0
	client := &partialRevokeTestClient{
		selectFunc: func(query string, callback func(clickhouseclient.Row) error) error {
			selectCount++
			// Exact lookup is first and returns no row. The second lookup lists
			// a database-level partial revoke that covers the requested column.
			if selectCount == 2 && strings.Contains(query, "`is_partial_revoke` = 1") {
				return callback(partialRevokeRow(
					"SELECT",
					new("analytics"),
					nil,
					nil,
					nil,
					nil,
					new("analyst"),
					false,
				))
			}
			return nil
		},
	}
	dbClient := &impl{clickhouseClient: client, readAfterWriteTimeout: time.Second}
	got, err := dbClient.CreatePartialRevoke(context.Background(), PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		TableName:       new("events"),
		ColumnName:      new("secret"),
		GranteeRoleName: new("analyst"),
	}, nil)
	if err != nil {
		t.Fatalf("CreatePartialRevoke() error = %v", err)
	}
	if got != nil {
		t.Fatalf("CreatePartialRevoke() = %#v, want nil for overlapping state", got)
	}
}

func TestCreatePartialRevoke_RequiresCoveringPositiveGrant(t *testing.T) {
	client := &partialRevokeTestClient{}
	dbClient := &impl{clickhouseClient: client, readAfterWriteTimeout: time.Second}
	got, err := dbClient.CreatePartialRevoke(context.Background(), PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		TableName:       new("events"),
		ColumnName:      new("secret"),
		GranteeRoleName: new("analyst"),
	}, nil)
	if err == nil {
		t.Fatal("CreatePartialRevoke() error = nil, want missing covering grant error")
	}
	if !strings.Contains(err.Error(), "no broader positive grant covers the target") {
		t.Fatalf("CreatePartialRevoke() error = %v, want missing covering grant error", err)
	}
	if got != nil {
		t.Fatalf("CreatePartialRevoke() = %#v, want nil", got)
	}
}

func TestGetPartialRevoke_UsesExactNegativeRightIdentity(t *testing.T) {
	client := &partialRevokeTestClient{}
	dbClient := &impl{clickhouseClient: client}
	partial := PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics_*"),
		TableName:       new("events_*"),
		ColumnName:      new("secret"),
		GranteeRoleName: new("analyst"),
		GrantOptionOnly: true,
	}

	got, err := dbClient.GetPartialRevoke(context.Background(), &partial, new("cluster"))
	if err != nil {
		t.Fatalf("GetPartialRevoke() error = %v", err)
	}
	if got != nil {
		t.Fatalf("GetPartialRevoke() = %#v, want nil", got)
	}
	query := client.selectQueries[0]
	for _, fragment := range []string{
		"FROM cluster('cluster', `system`.`grants`)",
		"`access_type` = 'SELECT'",
		"`is_partial_revoke` = 1",
		"`grant_option` = true",
		"`database` = 'analytics_'",
		"`table` = 'events_'",
		"`column` = 'secret'",
		"`role_name` = 'analyst'",
	} {
		if !strings.Contains(query, fragment) {
			t.Errorf("query %q does not contain %q", query, fragment)
		}
	}
}

func TestDeletePartialRevoke_RestoresOnlyTarget(t *testing.T) {
	selectCount := 0
	client := &partialRevokeTestClient{
		selectFunc: func(_ string, callback func(clickhouseclient.Row) error) error {
			selectCount++
			if selectCount == 1 {
				return callback(accessTypeRow("SELECT"))
			}
			return callback(positiveGrantRow(
				"SELECT",
				new("analytics"),
				new("events"),
				nil,
				nil,
				nil,
				new("analyst"),
				false,
			))
		},
	}
	dbClient := &impl{clickhouseClient: client}
	err := dbClient.DeletePartialRevoke(context.Background(), PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		TableName:       new("events"),
		ColumnName:      new("secret"),
		GranteeRoleName: new("analyst"),
	}, nil)
	if err != nil {
		t.Fatalf("DeletePartialRevoke() error = %v", err)
	}
	want := "GRANT SELECT(`secret`) ON `analytics`.`events` TO `analyst`;"
	if client.execQueries[0] != want {
		t.Errorf("Exec query = %q, want %q", client.execQueries[0], want)
	}
}

func TestDeletePartialRevoke_GrantOptionOnly(t *testing.T) {
	selectCount := 0
	client := &partialRevokeTestClient{
		selectFunc: func(_ string, callback func(clickhouseclient.Row) error) error {
			selectCount++
			if selectCount == 1 {
				return callback(accessTypeRow("SELECT"))
			}
			return callback(positiveGrantRow(
				"SELECT",
				new("analytics"),
				nil,
				nil,
				nil,
				new("reporter"),
				nil,
				true,
			))
		},
	}
	dbClient := &impl{clickhouseClient: client}
	err := dbClient.DeletePartialRevoke(context.Background(), PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		GranteeUserName: new("reporter"),
		GrantOptionOnly: true,
	}, nil)
	if err != nil {
		t.Fatalf("DeletePartialRevoke() error = %v", err)
	}
	want := "GRANT SELECT ON `analytics`.* TO `reporter` WITH GRANT OPTION;"
	if client.execQueries[0] != want {
		t.Errorf("Exec query = %q, want %q", client.execQueries[0], want)
	}
}

func TestDeletePartialRevoke_AlreadyAbsentIsIdempotent(t *testing.T) {
	client := &partialRevokeTestClient{}
	dbClient := &impl{clickhouseClient: client}
	err := dbClient.DeletePartialRevoke(context.Background(), PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		GranteeRoleName: new("analyst"),
	}, nil)
	if err != nil {
		t.Fatalf("DeletePartialRevoke() error = %v", err)
	}
	if len(client.execQueries) != 0 {
		t.Fatalf("Exec query count = %d, want 0", len(client.execQueries))
	}
}

func TestDeletePartialRevoke_RefusesToGrantWhenCoveringPositiveGrantIsGone(t *testing.T) {
	selectCount := 0
	client := &partialRevokeTestClient{
		selectFunc: func(_ string, callback func(clickhouseclient.Row) error) error {
			selectCount++
			if selectCount == 1 {
				return callback(accessTypeRow("SELECT"))
			}
			// The partial row exists, but there are no positive grants.
			return nil
		},
	}
	dbClient := &impl{clickhouseClient: client}
	err := dbClient.DeletePartialRevoke(context.Background(), PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		TableName:       new("events"),
		ColumnName:      new("secret"),
		GranteeRoleName: new("analyst"),
	}, nil)
	if err == nil {
		t.Fatal("DeletePartialRevoke() error = nil, want fail-closed error")
	}
	if !strings.Contains(err.Error(), "refusing to delete partial revoke") {
		t.Fatalf("DeletePartialRevoke() error = %v, want refusal", err)
	}
	if len(client.execQueries) != 0 {
		t.Fatalf("Exec query count = %d, want 0", len(client.execQueries))
	}
}

func TestDeletePartialRevoke_GrantOptionOnlyRequiresCoveringGrantOption(t *testing.T) {
	selectCount := 0
	client := &partialRevokeTestClient{
		selectFunc: func(_ string, callback func(clickhouseclient.Row) error) error {
			selectCount++
			if selectCount == 1 {
				return callback(accessTypeRow("SELECT"))
			}
			// A basic positive grant does not make WITH GRANT OPTION safe.
			return callback(positiveGrantRow(
				"SELECT",
				new("analytics"),
				nil,
				nil,
				nil,
				new("reporter"),
				nil,
				false,
			))
		},
	}
	dbClient := &impl{clickhouseClient: client}
	err := dbClient.DeletePartialRevoke(context.Background(), PartialRevoke{
		AccessType:      "SELECT",
		DatabaseName:    new("analytics"),
		GranteeUserName: new("reporter"),
		GrantOptionOnly: true,
	}, nil)
	if err == nil {
		t.Fatal("DeletePartialRevoke() error = nil, want fail-closed error")
	}
	if len(client.execQueries) != 0 {
		t.Fatalf("Exec query count = %d, want 0", len(client.execQueries))
	}
}

func TestCoversPartialRevoke(t *testing.T) {
	tests := []struct {
		name     string
		broader  PartialRevoke
		narrower PartialRevoke
		want     bool
	}{
		{
			name:     "full database revoke covers column revoke",
			broader:  PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics")},
			narrower: PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics"), TableName: new("events"), ColumnName: new("secret")},
			want:     true,
		},
		{
			name:     "full revoke covers grant option only",
			broader:  PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics")},
			narrower: PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics"), GrantOptionOnly: true},
			want:     true,
		},
		{
			name:     "grant option only does not cover full revoke",
			broader:  PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics"), GrantOptionOnly: true},
			narrower: PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics")},
			want:     false,
		},
		{
			name:     "column revoke does not cover table revoke",
			broader:  PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics"), TableName: new("events"), ColumnName: new("secret")},
			narrower: PartialRevoke{AccessType: "SELECT", DatabaseName: new("analytics"), TableName: new("events")},
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CoversPartialRevoke(tt.broader, tt.narrower); got != tt.want {
				t.Errorf("CoversPartialRevoke() = %v, want %v", got, tt.want)
			}
		})
	}
}
