package settingsprofileassociation

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SettingsProfileAssociation struct {
	ClusterName       types.String `tfsdk:"cluster_name"`
	SettingsProfileID types.String `tfsdk:"settings_profile_id"`
	RoleID            types.String `tfsdk:"role_id"`
	UserID            types.String `tfsdk:"user_id"`
}
