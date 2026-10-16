package dbops

import (
	"context"
	"testing"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/nilcompare"
)

const (
	versionSQL              = "SELECT `value` FROM `system`.`build_options` WHERE (`name` = 'VERSION_FULL');"
	granteeGrantsSQL        = "SELECT toString(`access_type`) AS `access_type`, `database`, `table`, `column`, nullIf(access_object, '') AS `access_object_nullable`, `user_name`, `role_name`, `grant_option` FROM `system`.`grants` WHERE ((`is_partial_revoke` = 0 AND `user_name` = 'app'));"
	granteeGrantsWithWcdSQL = "SELECT toString(`access_type`) AS `access_type`, `database`, `table`, `column`, nullIf(access_object, '') AS `access_object_nullable`, `user_name`, `role_name`, `grant_option`, `is_wildcard` FROM `system`.`grants` WHERE ((`is_partial_revoke` = 0 AND `user_name` = 'app'));"
)

func grantRow(database, table, accessObject *string, wildcard bool) map[string]any {
	return map[string]any{
		"access_type":            "SELECT",
		"database":               database,
		"table":                  table,
		"column":                 (*string)(nil),
		"access_object_nullable": accessObject,
		"user_name":              new("app"),
		"role_name":              (*string)(nil),
		"grant_option":           false,
		"is_wildcard":            wildcard,
	}
}

func TestGetAllGrantsForGrantee(t *testing.T) {
	type object struct{ database, table, accessObject *string }

	tests := []struct {
		name      string
		version   string
		grantsSQL string
		rows      []map[string]any
		want      []object
	}{
		{
			name:      "restores the wildcard on the deepest dimension",
			version:   "26.9.4.3",
			grantsSQL: granteeGrantsWithWcdSQL,
			rows: []map[string]any{
				grantRow(new("db"), nil, nil, true),
				grantRow(new("db"), new("t"), nil, true),
				grantRow(nil, nil, new("bob"), true),
				grantRow(new("db"), new("x"), nil, false),
			},
			want: []object{
				{database: new("db*")},
				{database: new("db"), table: new("t*")},
				{accessObject: new("bob*")},
				{database: new("db"), table: new("x")},
			},
		},
		{
			name:      "treats every grant as a wildcard before 26.7",
			version:   "26.6.8.7",
			grantsSQL: granteeGrantsSQL,
			rows: []map[string]any{
				grantRow(nil, nil, nil, false),
				grantRow(new("db"), nil, nil, false),
				grantRow(new("db"), new("t"), nil, false),
				grantRow(nil, nil, new("bob"), false),
			},
			want: []object{
				{},
				{database: new("db*")},
				{database: new("db"), table: new("t*")},
				{accessObject: new("bob*")},
			},
		},
	}

	str := func(s *string) string {
		if s == nil {
			return "<nil>"
		}
		return *s
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, fake := newTestClient(t, []step{
				{wantSQL: versionSQL, rows: []map[string]any{{"value": tc.version}}},
				{wantSQL: tc.grantsSQL, rows: tc.rows},
			})

			got, err := client.GetAllGrantsForGrantee(context.Background(), new("app"), nil, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(fake.steps) != 0 {
				t.Errorf("script not fully consumed, %d steps left", len(fake.steps))
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d grants, want %d", len(got), len(tc.want))
			}
			for idx, w := range tc.want {
				g := got[idx]
				if !nilcompare.NilCompare(g.DatabaseName, w.database) || !nilcompare.NilCompare(g.TableName, w.table) || !nilcompare.NilCompare(g.AccessObject, w.accessObject) {
					t.Errorf("grant %d = %s.%s %s, want %s.%s %s", idx, str(g.DatabaseName), str(g.TableName), str(g.AccessObject), str(w.database), str(w.table), str(w.accessObject))
				}
			}
		})
	}
}
