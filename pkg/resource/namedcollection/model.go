package namedcollection

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type NamedCollection struct {
	ClusterName        types.String `tfsdk:"cluster_name"`
	Name               types.String `tfsdk:"name"`
	Keys               types.Map    `tfsdk:"keys"`
	OverridableKeys    types.Set    `tfsdk:"overridable_keys"`
	NotOverridableKeys types.Set    `tfsdk:"not_overridable_keys"`
}
