package maskingpolicy

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type MaskingPolicy struct {
	Name             types.String `tfsdk:"name"`
	Database         types.String `tfsdk:"database_name"`
	Table            types.String `tfsdk:"table_name"`
	Masks            types.Map    `tfsdk:"masks"`
	WhereExpression  types.String `tfsdk:"where_expression"`
	GranteeUserNames types.List   `tfsdk:"grantee_user_names"`
	GranteeRoleNames types.List   `tfsdk:"grantee_role_names"`
	GranteeAll       types.Bool   `tfsdk:"grantee_all"`
	GranteeAllExcept types.List   `tfsdk:"grantee_all_except"`
	Priority         types.Int64  `tfsdk:"priority"`
}
