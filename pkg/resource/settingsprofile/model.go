package settingsprofile

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type SettingsProfile struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	InheritFrom types.List   `tfsdk:"inherit_from"`
}
