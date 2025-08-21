package setting

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed setting.md
var settingResourceDescription string

var (
	_ resource.Resource               = &Resource{}
	_ resource.ResourceWithConfigure  = &Resource{}
	_ resource.ResourceWithModifyPlan = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_setting"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the cluster to create the resource into. If omitted, resource will be created on the replica hit by the query.\nThis field must be left null when using a ClickHouse Cloud cluster.\nWhen using a self hosted ClickHouse instance, this field should only be set when there is more than one replica and you are not using 'replicated' storage for user_directory.\n",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"settings_profile_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the settings profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"name": schema.StringAttribute{
				Description: "Name of the setting",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"value": schema.StringAttribute{
				Description: "Value for the setting",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("min"),
						path.MatchRoot("max"),
					),
				},
			},
			"min": schema.StringAttribute{
				Description: "Min Value for the setting",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("value"),
						path.MatchRoot("max"),
					),
				},
			},
			"max": schema.StringAttribute{
				Description: "Max Value for the setting",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("value"),
						path.MatchRoot("min"),
					),
				},
			},
			"writability": schema.StringAttribute{
				Description: "Writability attribute for the setting",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"CONST",
						"WRITABLE",
						"CHANGEABLE_IN_READONLY",
					),
				},
			},
		},
		MarkdownDescription: settingResourceDescription,
	}
}

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// If the entire plan is null, the resource is planned for destruction.
		return
	}

	if r.client != nil {
		isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Checking if service is using replicated storage",
				fmt.Sprintf("%+v\n", err),
			)
			return
		}

		if isReplicatedStorage {
			var config Setting
			diags := req.Config.Get(ctx, &config)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// Setting cannot specify 'cluster_name' or apply will fail.
			if !config.ClusterName.IsNull() {
				resp.Diagnostics.AddWarning(
					"Invalid configuration",
					"Your ClickHouse cluster is using Replicated storage, please remove the 'cluster_name' attribute from your Setting resource definition if you encounter any errors.",
				)
			}
		}
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(dbops.Client)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Setting
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	setting := dbops.Setting{
		Name:        plan.Name.ValueString(),
		Value:       plan.Value.ValueStringPointer(),
		Min:         plan.Min.ValueStringPointer(),
		Max:         plan.Max.ValueStringPointer(),
		Writability: plan.Writability.ValueStringPointer(),
	}

	createdSetting, err := r.client.CreateSetting(ctx, plan.SettingsProfileID.ValueString(), setting, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Setting",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	state := Setting{
		ClusterName:       plan.ClusterName,
		SettingsProfileID: plan.SettingsProfileID,
	}

	modelFromApiResponse(&state, *createdSetting)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state Setting
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settingsProfile, err := r.client.GetSetting(ctx, state.SettingsProfileID.ValueString(), state.Name.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Setting",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	if settingsProfile != nil {
		modelFromApiResponse(&state, *settingsProfile)

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.State.RemoveResource(ctx)
	}
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("Update of settingsprofilesetting resource is not supported")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state Setting
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSetting(ctx, state.SettingsProfileID.ValueString(), state.Name.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Setting",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}

func modelFromApiResponse(state *Setting, settingsProfile dbops.Setting) {
	state.Name = types.StringValue(settingsProfile.Name)
	state.Value = types.StringPointerValue(settingsProfile.Value)
	state.Min = types.StringPointerValue(settingsProfile.Min)
	state.Max = types.StringPointerValue(settingsProfile.Max)
	state.Writability = types.StringPointerValue(settingsProfile.Writability)
}
