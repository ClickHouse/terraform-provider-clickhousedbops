package setting

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Setting struct {
	ClusterName       types.String `tfsdk:"cluster_name"`
	SettingsProfileID types.String `tfsdk:"settings_profile_id"`
	Name              types.String `tfsdk:"name"`
	Value             types.String `tfsdk:"value"`
	Min               types.String `tfsdk:"min"`
	Max               types.String `tfsdk:"max"`
	Writability       types.String `tfsdk:"writability"`
}
