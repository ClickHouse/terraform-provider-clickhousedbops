package role

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Role struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
}
