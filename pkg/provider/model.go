package provider

// Model describes the provider data model.
type Model struct {
	Protocol   string     `tfsdk:"protocol"`
	Host       string     `tfsdk:"host"`
	Port       uint16     `tfsdk:"port"`
	AuthConfig AuthConfig `tfsdk:"auth_config"`
}

type AuthConfig struct {
	Strategy string  `tfsdk:"strategy"`
	Username string  `tfsdk:"username"`
	Password *string `tfsdk:"password"`
}
