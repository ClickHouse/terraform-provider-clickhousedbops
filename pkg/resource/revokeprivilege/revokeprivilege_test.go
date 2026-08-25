package revokeprivilege

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validConfig() RevokePrivilege {
	return RevokePrivilege{
		Privilege:       types.StringValue("SELECT"),
		Database:        types.StringValue("analytics"),
		Table:           types.StringValue("events"),
		Column:          types.StringValue("secret"),
		AccessObject:    types.StringNull(),
		GranteeUserName: types.StringNull(),
		GranteeRoleName: types.StringValue("analyst"),
		GrantOptionOnly: types.BoolValue(false),
		ClusterName:     types.StringNull(),
	}
}

func TestValidateScope(t *testing.T) {
	tests := []struct {
		name      string
		config    RevokePrivilege
		wantError bool
	}{
		{
			name:   "column partial revoke",
			config: validConfig(),
		},
		{
			name: "user-name access object",
			config: func() RevokePrivilege {
				config := validConfig()
				config.Privilege = types.StringValue("CREATE USER")
				config.Database = types.StringNull()
				config.Table = types.StringNull()
				config.Column = types.StringNull()
				config.AccessObject = types.StringValue("session_*")
				return config
			}(),
		},
		{
			name: "source partial revoke",
			config: func() RevokePrivilege {
				config := validConfig()
				config.Privilege = types.StringValue("READ")
				config.Database = types.StringNull()
				config.Table = types.StringNull()
				config.Column = types.StringNull()
				config.AccessObject = types.StringValue("S3")
				return config
			}(),
			wantError: true,
		},
		{
			name: "database target for global privilege",
			config: func() RevokePrivilege {
				config := validConfig()
				config.Privilege = types.StringValue("SHOW USERS")
				config.Table = types.StringNull()
				config.Column = types.StringNull()
				return config
			}(),
			wantError: true,
		},
		{
			name: "alias",
			config: func() RevokePrivilege {
				config := validConfig()
				config.Privilege = types.StringValue("SOURCE READ")
				return config
			}(),
			wantError: true,
		},
		{
			name: "privilege group",
			config: func() RevokePrivilege {
				config := validConfig()
				config.Privilege = types.StringValue("ALL")
				config.Database = types.StringNull()
				config.Table = types.StringNull()
				config.Column = types.StringNull()
				return config
			}(),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diagnostics diag.Diagnostics
			validateScope(tt.config, &diagnostics)
			if diagnostics.HasError() != tt.wantError {
				t.Errorf("validateScope() has error = %v, want %v; diagnostics: %v", diagnostics.HasError(), tt.wantError, diagnostics)
			}
		})
	}
}
