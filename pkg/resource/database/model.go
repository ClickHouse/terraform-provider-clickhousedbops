package database

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Database struct {
	ClusterName types.String `tfsdk:"cluster_name"`
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Comment     types.String `tfsdk:"comment"`
}
