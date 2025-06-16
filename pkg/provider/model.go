package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model describes the provider data model.
type Model struct {
	Protocol   types.String `tfsdk:"protocol"`
	Host       types.String `tfsdk:"host"`
	Port       types.Number `tfsdk:"port"`
	AuthConfig AuthConfig   `tfsdk:"auth_config"`
}

type AuthConfig struct {
	Strategy string  `tfsdk:"strategy"`
	Username string  `tfsdk:"username"`
	Password *string `tfsdk:"password"`
}
