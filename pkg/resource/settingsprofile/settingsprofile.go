package settingsprofile

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed settingsprofile.md
var settingsProfileResourceDescription string

var (
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithConfigure   = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
	_ resource.ResourceWithModifyPlan  = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings_profile"
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
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "ID of the settings profile",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the settings profile",
			},
			"inherit_from": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "List of setting profile names to inherit from",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
		},
		MarkdownDescription: settingsProfileResourceDescription,
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
			var config SettingsProfile
			diags := req.Config.Get(ctx, &config)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// SettingsProfile cannot specify 'cluster_name' or apply will fail.
			if !config.ClusterName.IsNull() {
				resp.Diagnostics.AddWarning(
					"Invalid configuration",
					"Your ClickHouse cluster is using Replicated storage, please remove the 'cluster_name' attribute from your SettingsProfile resource definition if you encounter any errors.",
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
	var plan SettingsProfile
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inherit := make([]string, 0)

	resp.Diagnostics.Append(plan.InheritFrom.ElementsAs(ctx, &inherit, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile := dbops.SettingsProfile{
		Name:        plan.Name.ValueString(),
		InheritFrom: inherit,
	}

	createdSettingsProfile, err := r.client.CreateSettingsProfile(ctx, profile, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse SettingsProfile",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	state := SettingsProfile{
		ClusterName: plan.ClusterName,
	}

	modelFromApiResponse(&state, *createdSettingsProfile)

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SettingsProfile
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	settingsProfile, err := r.client.GetSettingsProfile(ctx, state.ID.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse SettingsProfile",
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
	var plan, state SettingsProfile
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	inherit := make([]string, 0)
	resp.Diagnostics.Append(plan.InheritFrom.ElementsAs(ctx, &inherit, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profile := dbops.SettingsProfile{
		ID:          state.ID.ValueString(),
		Name:        plan.Name.ValueString(),
		InheritFrom: inherit,
	}

	editedProfile, err := r.client.UpdateSettingsProfile(ctx, profile, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating ClickHouse SettingsProfile",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
	if editedProfile != nil {
		modelFromApiResponse(&state, *editedProfile)

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.State.RemoveResource(ctx)
	}
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SettingsProfile
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSettingsProfile(ctx, state.ID.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse SettingsProfile",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// req.ID can either be in the form <cluster name>:<setting profile ref> or just <setting profile ref>
	// setting profile ref can either be the settings profile's name or the UUID

	// Check if cluster name is specified
	ref := req.ID
	var clusterName *string
	if strings.Contains(req.ID, ":") {
		clusterName = &strings.Split(req.ID, ":")[0]
		ref = strings.Split(req.ID, ":")[1]
	}

	// Check if ref is a UUID
	_, err := uuid.Parse(ref)
	if err != nil {
		// Failed parsing UUID, try importing using the database name
		settingsProfile, err := r.client.FindSettingsProfileByName(ctx, ref, clusterName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Cannot find settings profile",
				fmt.Sprintf("%+v\n", err),
			)
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), settingsProfile.ID)...)
	} else {
		// User passed a UUID
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ref)...)
	}

	if clusterName != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	}
}

func modelFromApiResponse(state *SettingsProfile, settingsProfile dbops.SettingsProfile) {
	state.ID = types.StringValue(settingsProfile.ID)
	state.Name = types.StringValue(settingsProfile.Name)

	if len(settingsProfile.InheritFrom) > 0 {
		values := make([]attr.Value, 0)
		for _, i := range settingsProfile.InheritFrom {
			values = append(values, types.StringValue(i))
		}

		state.InheritFrom, _ = types.ListValue(types.StringType, values)
	} else {
		state.InheritFrom = types.ListNull(types.StringType)
	}
}
