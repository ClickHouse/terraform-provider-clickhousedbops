package settingsprofileassociation

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SettingsProfileAssociation struct {
	ClusterName         types.String `tfsdk:"cluster_name"`
	SettingsProfileName types.String `tfsdk:"settings_profile_name"`
	RoleName            types.String `tfsdk:"role_name"`
	UserName            types.String `tfsdk:"user_name"`
}
