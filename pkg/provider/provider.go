package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/pkg/project"
)

const (
	protocolNative       = "native"
	protocolNativeSecure = "nativesecure"

	authStrategyPassword = "password"
)

var (
	availableProtocols      = []string{protocolNative, protocolNativeSecure}
	availableAuthStrategies = []string{authStrategyPassword}
)

// Ensure Provider satisfies various provider interfaces.
var _ provider.Provider = &Provider{}

// Provider defines the provider implementation.
type Provider struct{}

func (p *Provider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "clickhousedbops"
	resp.Version = project.Version()
}

func (p *Provider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"protocol": schema.StringAttribute{
				Required:    true,
				Description: fmt.Sprintf("The protocol to use to connect to clickhouse instance. Valid options are: %s", strings.Join(availableProtocols, ", ")),
				Validators: []validator.String{
					stringvalidator.OneOf(availableProtocols...),
				},
			},
			"host": schema.StringAttribute{
				Required:    true,
				Description: "The hostname to use to connect to the clickhouse instance",
			},
			"port": schema.NumberAttribute{
				Required:    true,
				Description: "The port to use to connect to the clickhouse instance",
			},
			"auth_config": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"strategy": schema.StringAttribute{
						Required:    true,
						Description: "The authentication method to use",
						Validators: []validator.String{
							stringvalidator.OneOf(availableAuthStrategies...),
						},
					},
					"username": schema.StringAttribute{
						Required:    true,
						Description: "The username to use to authenticate to ClickHouse",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"password": schema.StringAttribute{
						Optional:    true,
						Description: "The password to use to authenticate to ClickHouse",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
				},
				Required:    true,
				Description: "Authentication configuration",
			},
		},
	}
}

func (p *Provider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data Model
	var err error

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	{
		switch data.Protocol {
		case protocolNative:
			fallthrough
		case protocolNativeSecure:
			var auth *clickhouseclient.UserPasswordAuth
			switch data.AuthConfig.Strategy {
			case authStrategyPassword:
				auth = &clickhouseclient.UserPasswordAuth{
					Username: data.AuthConfig.Username,
				}

				if data.AuthConfig.Password != nil {
					auth.Password = *data.AuthConfig.Password
				}

				valid, errorStrings := auth.ValidateConfig()
				if !valid {
					resp.Diagnostics.AddError("invalid configuration", fmt.Sprintf("invalid authentication strategy configuration. %s", strings.Join(errorStrings, ", ")))
				}
			default:
				resp.Diagnostics.AddError("invalid configuration", fmt.Sprintf("invalid authentication strategy %q. %s protocol only supports %q", data.AuthConfig.Strategy, protocolNative, authStrategyPassword))
				return
			}

			_, err = clickhouseclient.NewNativeClient(clickhouseclient.NativeClientConfig{
				Host:             data.Host,
				Port:             data.Port,
				UserPasswordAuth: auth,
				EnableTLS:        data.Protocol == protocolNativeSecure,
			})
		}
	}

	if err != nil {
		resp.Diagnostics.AddError("error initializing query runner", fmt.Sprintf("%+v\n", err))
	}
}

func (p *Provider) Resources(ctx context.Context) []func() tfresource.Resource {
	return []func() tfresource.Resource{}
}

func (p *Provider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func New() func() provider.Provider {
	return func() provider.Provider {
		return &Provider{}
	}
}
