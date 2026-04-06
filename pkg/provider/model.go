package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Model describes the provider data model.
type Model struct {
	Protocol              types.String `tfsdk:"protocol"`
	Host                  types.String `tfsdk:"host"`
	Port                  types.Int32  `tfsdk:"port"`
	AuthConfig            AuthConfig   `tfsdk:"auth_config"`
	TLSConfig             *TLSConfig   `tfsdk:"tls_config"`
	ReadAfterWriteTimeout types.Int64  `tfsdk:"read_after_write_timeout"`
}

type AuthConfig struct {
	Strategy types.String `tfsdk:"strategy"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

type TLSConfig struct {
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
	CACert             types.String `tfsdk:"ca_cert"`
}
