package grantrole

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GrantRole struct {
	ClusterName     types.String `tfsdk:"cluster_name"`
	RoleName        types.String `tfsdk:"role_name"`
	GranteeUserName types.String `tfsdk:"grantee_user_name"`
	GranteeRoleName types.String `tfsdk:"grantee_role_name"`
	AdminOption     types.Bool   `tfsdk:"admin_option"`
}
