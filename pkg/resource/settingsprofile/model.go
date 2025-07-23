package settingsprofile

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type SettingsProfile struct {
	ClusterName    types.String `tfsdk:"cluster_name"`
	Name           types.String `tfsdk:"name"`
	Settings       types.List   `tfsdk:"settings"`
	InheritProfile types.String `tfsdk:"inherit_profile"`
}

type Setting struct {
	Name        types.String `tfsdk:"name"`
	Value       types.String `tfsdk:"value"`
	Min         types.String `tfsdk:"min"`
	Max         types.String `tfsdk:"max"`
	Writability types.String `tfsdk:"writability"`
}

func (i Setting) ObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"name":        types.StringType,
			"value":       types.StringType,
			"min":         types.StringType,
			"max":         types.StringType,
			"writability": types.StringType,
		},
	}
}

func (i Setting) ObjectValue() basetypes.ObjectValue {
	return types.ObjectValueMust(i.ObjectType().AttrTypes, map[string]attr.Value{
		"name":        i.Name,
		"value":       i.Value,
		"min":         i.Min,
		"max":         i.Max,
		"writability": i.Writability,
	})
}
