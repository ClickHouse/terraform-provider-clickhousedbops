package revokeprivilege

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

type RevokePrivilege struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	Privilege       types.String `tfsdk:"privilege_name"`
	Database        types.String `tfsdk:"database_name"`
	Table           types.String `tfsdk:"table_name"`
	Column          types.String `tfsdk:"column_name"`
	AccessObject    types.String `tfsdk:"access_object"`
	GranteeUserName types.String `tfsdk:"grantee_user_name"`
	GranteeRoleName types.String `tfsdk:"grantee_role_name"`
	GrantOptionOnly types.Bool   `tfsdk:"grant_option_only"`
}

func (r RevokePrivilege) toPartialRevoke() dbops.PartialRevoke {
	return dbops.PartialRevoke{
		AccessType:      r.Privilege.ValueString(),
		DatabaseName:    r.Database.ValueStringPointer(),
		TableName:       r.Table.ValueStringPointer(),
		ColumnName:      r.Column.ValueStringPointer(),
		AccessObject:    r.AccessObject.ValueStringPointer(),
		GranteeUserName: r.GranteeUserName.ValueStringPointer(),
		GranteeRoleName: r.GranteeRoleName.ValueStringPointer(),
		GrantOptionOnly: r.GrantOptionOnly.ValueBool(),
	}
}

func toState(r dbops.PartialRevoke, clusterName types.String) RevokePrivilege {
	return RevokePrivilege{
		ClusterName:     clusterName,
		Privilege:       types.StringValue(r.AccessType),
		Database:        types.StringPointerValue(r.DatabaseName),
		Table:           types.StringPointerValue(r.TableName),
		Column:          types.StringPointerValue(r.ColumnName),
		AccessObject:    types.StringPointerValue(r.AccessObject),
		GranteeUserName: types.StringPointerValue(r.GranteeUserName),
		GranteeRoleName: types.StringPointerValue(r.GranteeRoleName),
		GrantOptionOnly: types.BoolValue(r.GrantOptionOnly),
	}
}
