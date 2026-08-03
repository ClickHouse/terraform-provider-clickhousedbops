package grantprivilege

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
)

type GrantPrivilege struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	Privilege       types.String `tfsdk:"privilege_name"`
	Database        types.String `tfsdk:"database_name"`
	Table           types.String `tfsdk:"table_name"`
	Column          types.String `tfsdk:"column_name"`
	AccessObject    types.String `tfsdk:"access_object"`
	GranteeUserName types.String `tfsdk:"grantee_user_name"`
	GranteeRoleName types.String `tfsdk:"grantee_role_name"`
	GrantOption     types.Bool   `tfsdk:"grant_option"`
	CurrentGrants   types.Bool   `tfsdk:"current_grants"`
}

func (g GrantPrivilege) toGrant() dbops.GrantPrivilege {
	return dbops.GrantPrivilege{
		AccessType:          g.Privilege.ValueString(),
		ExpandedAccessTypes: grants.AllDescendants(grants.Parsed().Groups, g.Privilege.ValueString()),
		DatabaseName:        g.Database.ValueStringPointer(),
		TableName:           g.Table.ValueStringPointer(),
		ColumnName:          g.Column.ValueStringPointer(),
		AccessObject:        g.AccessObject.ValueStringPointer(),
		GranteeUserName:     g.GranteeUserName.ValueStringPointer(),
		GranteeRoleName:     g.GranteeRoleName.ValueStringPointer(),
		GrantOption:         g.GrantOption.ValueBool(),
		CurrentGrants:       g.CurrentGrants.ValueBool(),
	}
}

func (g GrantPrivilege) asGrant() grants.Grant {
	return grants.Grant{
		AccessType:   g.Privilege.ValueString(),
		Database:     g.Database.ValueStringPointer(),
		Table:        g.Table.ValueStringPointer(),
		Column:       g.Column.ValueStringPointer(),
		AccessObject: g.AccessObject.ValueStringPointer(),
		GrantOption:  g.GrantOption.ValueBool(),
	}
}

func toState(g dbops.GrantPrivilege, clusterName types.String) GrantPrivilege {
	return GrantPrivilege{
		ClusterName:     clusterName,
		Privilege:       types.StringValue(g.AccessType),
		Database:        types.StringPointerValue(g.DatabaseName),
		Table:           types.StringPointerValue(g.TableName),
		Column:          types.StringPointerValue(g.ColumnName),
		AccessObject:    types.StringPointerValue(g.AccessObject),
		GranteeUserName: types.StringPointerValue(g.GranteeUserName),
		GranteeRoleName: types.StringPointerValue(g.GranteeRoleName),
		GrantOption:     types.BoolValue(g.GrantOption),
	}
}
