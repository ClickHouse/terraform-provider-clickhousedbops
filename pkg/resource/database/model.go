package database

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type Database struct {
	ClusterName               types.String `tfsdk:"cluster_name"`
	UUID                      types.String `tfsdk:"uuid"`
	Name                      types.String `tfsdk:"name"`
	Comment                   types.String `tfsdk:"comment"`
	Engine                    types.String `tfsdk:"engine"`
	EngineArguments           types.List   `tfsdk:"engine_arguments"`
	EngineSettings            types.Map    `tfsdk:"engine_settings"`
	EngineParametersWO        types.Map    `tfsdk:"engine_parameters_wo"`
	EngineParametersWOVersion types.Int64  `tfsdk:"engine_parameters_wo_version"`
}
