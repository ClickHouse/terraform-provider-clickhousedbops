package grantprivilege

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

func Test_overlaps(t *testing.T) {
	tests := []struct {
		name     string
		current  GrantPrivilege
		existing dbops.GrantPrivilege
		want     bool
	}{
		// DatabaseName
		{
			name: "Database: Same value no wildcards",
			current: GrantPrivilege{
				Database: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				DatabaseName: toStrPtr("test"),
			},
			want: true,
		},
		{
			name: "Database: Different value no wildcards",
			current: GrantPrivilege{
				Database: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				DatabaseName: toStrPtr("test2"),
			},
			want: false,
		},
		{
			name: "Database: existing is wildcard, current is set",
			current: GrantPrivilege{
				Database: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				DatabaseName: nil,
			},
			want: true,
		},
		{
			name: "Database: existing is set, current is wildcard",
			current: GrantPrivilege{
				Database: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				DatabaseName: toStrPtr("test"),
			},
			want: false,
		},
		{
			name: "Database: current ends with wildcard, existing ends with wildcard and is overlapping",
			current: GrantPrivilege{
				Database: types.StringValue("test*"),
			},
			existing: dbops.GrantPrivilege{
				DatabaseName: toStrPtr("tes*"),
			},
			want: true,
		},
		{
			name: "Database: current ends with wildcard, existing is set with no wildcard",
			current: GrantPrivilege{
				Database: types.StringValue("test*"),
			},
			existing: dbops.GrantPrivilege{
				DatabaseName: toStrPtr("test"),
			},
			want: false,
		},
		// TableName
		{
			name: "Table: Same value no wildcards",
			current: GrantPrivilege{
				Table: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				TableName: toStrPtr("test"),
			},
			want: true,
		},
		{
			name: "Table: Different value no wildcards",
			current: GrantPrivilege{
				Table: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				TableName: toStrPtr("test2"),
			},
			want: false,
		},
		{
			name: "Table: existing is wildcard, current is set",
			current: GrantPrivilege{
				Table: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				TableName: nil,
			},
			want: true,
		},
		{
			name: "Table: existing is set, current is wildcard",
			current: GrantPrivilege{
				Table: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				TableName: toStrPtr("test"),
			},
			want: false,
		},
		{
			name: "Table: current ends with wildcard, existing ends with wildcard and is overlapping",
			current: GrantPrivilege{
				Table: types.StringValue("test*"),
			},
			existing: dbops.GrantPrivilege{
				TableName: toStrPtr("tes*"),
			},
			want: true,
		},
		{
			name: "Table: current ends with wildcard, existing is set with no wildcard",
			current: GrantPrivilege{
				Table: types.StringValue("test*"),
			},
			existing: dbops.GrantPrivilege{
				TableName: toStrPtr("test"),
			},
			want: false,
		},

		// Columns
		{
			name: "Column: current is set,  existing is nil",
			current: GrantPrivilege{
				Column: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				ColumnName: nil,
			},
			want: true,
		},
		{
			name: "Column: current is nil, existing is set",
			current: GrantPrivilege{
				Column: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				ColumnName: toStrPtr("test"),
			},
			want: false,
		},
		{
			name: "Column: both current and existing are nil",
			current: GrantPrivilege{
				Column: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				ColumnName: nil,
			},
			want: true,
		},
		{
			name: "Column: both current and existing are set and equal",
			current: GrantPrivilege{
				Column: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				ColumnName: toStrPtr("test"),
			},
			want: true,
		},
		{
			name: "Column: both current and existing are set but different",
			current: GrantPrivilege{
				Column: types.StringValue("test1"),
			},
			existing: dbops.GrantPrivilege{
				ColumnName: toStrPtr("test2"),
			},
			want: false,
		},

		// AccessType: group coverage and identity.
		{
			name: "AccessType: existing group covers a member",
			current: GrantPrivilege{
				Privilege: types.StringValue("CREATE TABLE"),
			},
			existing: dbops.GrantPrivilege{
				AccessType: "CREATE",
			},
			want: true,
		},
		{
			name: "AccessType: existing top-level group covers a deeper member",
			current: GrantPrivilege{
				Privilege: types.StringValue("CREATE TABLE"),
			},
			existing: dbops.GrantPrivilege{
				AccessType: "ALL",
			},
			want: true,
		},
		{
			name: "AccessType: existing leaf does not cover a group",
			current: GrantPrivilege{
				Privilege: types.StringValue("CREATE"),
			},
			existing: dbops.GrantPrivilege{
				AccessType: "CREATE TABLE",
			},
			want: false,
		},
		{
			name: "AccessType: unrelated privileges do not overlap",
			current: GrantPrivilege{
				Privilege: types.StringValue("SELECT"),
			},
			existing: dbops.GrantPrivilege{
				AccessType: "INSERT",
			},
			want: false,
		},

		// GrantOption
		{
			name: "Grant option needed but existing lacks it",
			current: GrantPrivilege{
				GrantOption: types.BoolValue(true),
			},
			existing: dbops.GrantPrivilege{
				GrantOption: false,
			},
			want: false,
		},
		{
			name: "Grant option needed and existing has it",
			current: GrantPrivilege{
				GrantOption: types.BoolValue(true),
			},
			existing: dbops.GrantPrivilege{
				GrantOption: true,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.current.Privilege.IsNull() {
				tt.current.Privilege = types.StringValue("SELECT")
				tt.existing.AccessType = "SELECT"
			}
			if got := overlaps(tt.current, tt.existing); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_scopeAttributesFor(t *testing.T) {
	tests := []struct {
		privilege string
		want      scopeAttributes
		wantAll   scopeAttributes
		supported bool
	}{
		// Leaves: own scope equals the descendant union.
		{"SELECT", scopeAttributes{database: true, table: true, column: true}, scopeAttributes{database: true, table: true, column: true}, true},
		{"CREATE DATABASE", scopeAttributes{database: true}, scopeAttributes{database: true}, true},
		{"CREATE USER", scopeAttributes{}, scopeAttributes{}, true},
		{"CREATE", scopeAttributes{}, scopeAttributes{database: true, table: true}, true},
		{"ACCESS MANAGEMENT", scopeAttributes{}, scopeAttributes{database: true, table: true}, true},
		{"ALL", scopeAttributes{}, scopeAttributes{database: true, table: true, column: true}, true},
		{"TABLE ENGINE", scopeAttributes{}, scopeAttributes{}, true},
		{"READ", scopeAttributes{}, scopeAttributes{}, true},
		{"CREATE NAMED COLLECTION", scopeAttributes{}, scopeAttributes{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.privilege, func(t *testing.T) {
			got, gotAll, ok := scopeAttributesFor(tt.privilege)
			if got != tt.want || gotAll != tt.wantAll || ok != tt.supported {
				t.Errorf("scopeAttributesFor(%q) = %+v, %+v, %v; want %+v, %+v, %v", tt.privilege, got, gotAll, ok, tt.want, tt.wantAll, tt.supported)
			}
		})
	}
}

func toStrPtr(s string) *string {
	return &s
}
