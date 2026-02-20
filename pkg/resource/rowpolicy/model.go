package rowpolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RowPolicy struct {
	ClusterName      types.String `tfsdk:"cluster_name"`
	Name             types.String `tfsdk:"name"`
	Database         types.String `tfsdk:"database_name"`
	Table            types.String `tfsdk:"table_name"`
	ForOperations    types.List   `tfsdk:"for_operations"`
	SelectFilter     types.String `tfsdk:"select_filter"`
	IsRestrictive    types.Bool   `tfsdk:"is_restrictive"`
	GranteeUserNames types.List   `tfsdk:"grantee_user_names"`
	GranteeRoleNames types.List   `tfsdk:"grantee_role_names"`
	GranteeAll       types.Bool   `tfsdk:"grantee_all"`
	GranteeAllExcept types.List   `tfsdk:"grantee_all_except"`
}
