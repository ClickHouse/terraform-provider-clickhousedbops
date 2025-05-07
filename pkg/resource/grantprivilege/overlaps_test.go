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

		// GranteeUserName
		{
			name: "GranteeUserName: both set and equal",
			current: GrantPrivilege{
				GranteeUserName: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				GranteeUserName: toStrPtr("test"),
			},
			want: true,
		},
		{
			name: "GranteeUserName: both set and different",
			current: GrantPrivilege{
				GranteeUserName: types.StringValue("test1"),
			},
			existing: dbops.GrantPrivilege{
				GranteeUserName: toStrPtr("test2"),
			},
			want: false,
		},
		{
			name: "GranteeUserName: both nil",
			current: GrantPrivilege{
				GranteeUserName: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				GranteeUserName: nil,
			},
			want: true,
		},
		{
			name: "GranteeUserName: current nil, existing set",
			current: GrantPrivilege{
				GranteeUserName: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				GranteeUserName: toStrPtr("test"),
			},
			want: false,
		},
		{
			name: "GranteeUserName: current set, existing nil",
			current: GrantPrivilege{
				GranteeUserName: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				GranteeUserName: nil,
			},
			want: false,
		},

		// GranteeRoleName
		{
			name: "GranteeRoleName: both set and equal",
			current: GrantPrivilege{
				GranteeRoleName: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				GranteeRoleName: toStrPtr("test"),
			},
			want: true,
		},
		{
			name: "GranteeRoleName: both set and different",
			current: GrantPrivilege{
				GranteeRoleName: types.StringValue("test1"),
			},
			existing: dbops.GrantPrivilege{
				GranteeRoleName: toStrPtr("test2"),
			},
			want: false,
		},
		{
			name: "GranteeRoleName: both nil",
			current: GrantPrivilege{
				GranteeRoleName: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				GranteeRoleName: nil,
			},
			want: true,
		},
		{
			name: "GranteeRoleName: current nil, existing set",
			current: GrantPrivilege{
				GranteeRoleName: types.StringNull(),
			},
			existing: dbops.GrantPrivilege{
				GranteeRoleName: toStrPtr("test"),
			},
			want: false,
		},
		{
			name: "GranteeRoleName: current set, existing nil",
			current: GrantPrivilege{
				GranteeRoleName: types.StringValue("test"),
			},
			existing: dbops.GrantPrivilege{
				GranteeRoleName: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := overlaps(tt.current, tt.existing); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func toStrPtr(s string) *string {
	return &s
}
