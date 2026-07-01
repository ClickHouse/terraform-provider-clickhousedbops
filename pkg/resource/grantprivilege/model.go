package grantprivilege

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

type GrantPrivilege struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	Privilege       types.String `tfsdk:"privilege_name"`
	Database        types.String `tfsdk:"database_name"`
	Table           types.String `tfsdk:"table_name"`
	Column          types.String `tfsdk:"column_name"`
	GranteeUserName types.String `tfsdk:"grantee_user_name"`
	GranteeRoleName types.String `tfsdk:"grantee_role_name"`
	GrantOption     types.Bool   `tfsdk:"grant_option"`
}

func (g GrantPrivilege) toGrant() dbops.GrantPrivilege {
	return dbops.GrantPrivilege{
		AccessType:          g.Privilege.ValueString(),
		ExpandedAccessTypes: AllDescendants(parsedGrants().Groups, g.Privilege.ValueString()),
		DatabaseName:        g.Database.ValueStringPointer(),
		TableName:           g.Table.ValueStringPointer(),
		ColumnName:          g.Column.ValueStringPointer(),
		GranteeUserName:     g.GranteeUserName.ValueStringPointer(),
		GranteeRoleName:     g.GranteeRoleName.ValueStringPointer(),
		GrantOption:         g.GrantOption.ValueBool(),
	}
}

func toState(g dbops.GrantPrivilege, clusterName types.String) GrantPrivilege {
	return GrantPrivilege{
		ClusterName:     clusterName,
		Privilege:       types.StringValue(g.AccessType),
		Database:        types.StringPointerValue(g.DatabaseName),
		Table:           types.StringPointerValue(g.TableName),
		Column:          types.StringPointerValue(g.ColumnName),
		GranteeUserName: types.StringPointerValue(g.GranteeUserName),
		GranteeRoleName: types.StringPointerValue(g.GranteeRoleName),
		GrantOption:     types.BoolValue(g.GrantOption),
	}
}
