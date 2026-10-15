package dbops

import (
	"context"
	"slices"
	"testing"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
)

const privilegesSQL = "SELECT toString(`privilege`) AS `privilege`, toString(`level`) AS `level`, toString(`parent_group`) AS `parent_group` FROM `system`.`privileges`;"

func TestGetPrivileges(t *testing.T) {
	client, fake := newTestClient(t, []step{{wantSQL: privilegesSQL, rows: []map[string]any{
		{"privilege": "ALTER UPDATE", "level": new("COLUMN"), "parent_group": new("ALTER TABLE")},
		{"privilege": "ALTER TABLE", "level": (*string)(nil), "parent_group": new("ALL")},
		{"privilege": "ALL", "level": (*string)(nil), "parent_group": (*string)(nil)},
	}}})

	catalog, err := client.GetPrivileges(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := grants.AllDescendants(catalog.Groups, "ALL"); !slices.Equal(got, []string{"ALL", "ALTER TABLE", "ALTER UPDATE"}) {
		t.Errorf("descendants of ALL = %v", got)
	}
	if got := catalog.Scopes["ALTER UPDATE"]; got != "COLUMN" {
		t.Errorf("scope of ALTER UPDATE = %q, want COLUMN", got)
	}
	if _, ok := catalog.Scopes["ALL"]; ok {
		t.Errorf("ALL has a scope")
	}

	// The scripted client fails on any further query.
	if _, err := client.GetPrivileges(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fake.steps) != 0 {
		t.Errorf("script not fully consumed, %d steps left", len(fake.steps))
	}
}
