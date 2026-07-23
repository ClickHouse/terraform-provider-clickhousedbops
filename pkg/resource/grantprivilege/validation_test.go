package grantprivilege

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateScopeParameterizedTargets(t *testing.T) {
	tests := []struct {
		name       string
		privilege  string
		object     *string
		filter     *string
		database   *string
		wantErrors int
	}{
		{name: "table engine name", privilege: "TABLE ENGINE", object: new("Distributed")},
		{name: "all table engines", privilege: "TABLE ENGINE"},
		{name: "named source", privilege: "READ", object: new("S3")},
		{name: "filtered source", privilege: "READ", object: new("URL"), filter: new(`https://example\.com/.*`)},
		{name: "named collection", privilege: "NAMED COLLECTION", object: new("production_s3")},
		{name: "filter rejected for table engine", privilege: "TABLE ENGINE", object: new("Distributed"), filter: new(".*"), wantErrors: 1},
		{name: "source cannot use database target", privilege: "READ", database: new("default"), wantErrors: 1},
		{name: "database privilege cannot use access object", privilege: "SELECT", object: new("S3"), wantErrors: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GrantPrivilege{
				Privilege:          types.StringValue(tt.privilege),
				AccessObject:       types.StringPointerValue(tt.object),
				AccessObjectFilter: types.StringPointerValue(tt.filter),
				Database:           types.StringPointerValue(tt.database),
				Table:              types.StringNull(),
				Column:             types.StringNull(),
				ClusterName:        types.StringNull(),
				CurrentGrants:      types.BoolValue(false),
			}
			var diagnostics diag.Diagnostics
			validateScope(config, &diagnostics)

			if got := diagnostics.ErrorsCount(); got != tt.wantErrors {
				t.Fatalf("validateScope() error count = %d, want %d: %#v", got, tt.wantErrors, diagnostics)
			}
		})
	}
}
