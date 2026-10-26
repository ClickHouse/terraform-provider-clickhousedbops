package setting

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// Private-state entry holding the setting as ClickHouse stored it, so Read can tell drift from ClickHouse reformatting the configured values.
const storedValuesKey = "stored_values"

type storedValues struct {
	Value       *string `json:"value"`
	Min         *string `json:"min"`
	Max         *string `json:"max"`
	Writability *string `json:"writability"`
}

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

func (r *Resource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
			}),
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
		var config Setting
		diags := req.Config.Get(ctx, &config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Only check replicated storage when cluster_name is set, to avoid
		// unnecessary connections (e.g. during terraform plan -refresh=false).
		if !config.ClusterName.IsNull() {
			isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Could not check if service is using replicated storage",
					fmt.Sprintf("Skipping validation. If you are using replicated storage, please remove the 'cluster_name' attribute from your resource definition. Error: %+v", err),
				)
				return
			}

			// Setting cannot specify 'cluster_name' or apply will fail.
			if isReplicatedStorage {
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

	createTimeout, diags := plan.Timeouts.Create(ctx, 1*time.Minute)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	createdSetting, err := r.client.CreateSetting(ctx, plan.SettingsProfileID.ValueString(), setting, plan.ClusterName.ValueStringPointer(), createTimeout)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Setting",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	if createdSetting == nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Setting",
			"Created setting returned nil without error",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(setStoredValues(ctx, resp.Private, *createdSetting)...)
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

	if settingsProfile == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	stored, diags := getStoredValues(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// State written before the private entry existed holds the values as ClickHouse stored them.
	if stored == nil {
		stored = &storedValues{Value: state.Value.ValueStringPointer(), Min: state.Min.ValueStringPointer(), Max: state.Max.ValueStringPointer(), Writability: state.Writability.ValueStringPointer()}
	}

	if !reflect.DeepEqual(storedValuesOf(*settingsProfile), *stored) {
		modelFromApiResponse(&state, *settingsProfile)
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
		if resp.Diagnostics.HasError() {
			return
		}

		resp.Diagnostics.Append(setStoredValues(ctx, resp.Private, *settingsProfile)...)
	}
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state Setting
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Timeouts = plan.Timeouts
	if !reflect.DeepEqual(plan, state) {
		resp.Diagnostics.AddError("Error Updating ClickHouse Setting", "Only 'timeouts' can be updated in place")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
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

func storedValuesOf(setting dbops.Setting) storedValues {
	return storedValues{Value: setting.Value, Min: setting.Min, Max: setting.Max, Writability: setting.Writability}
}

// The framework's concrete private state type is in an internal package, so it can't be named directly.
type privateState interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func setStoredValues(ctx context.Context, private privateState, setting dbops.Setting) diag.Diagnostics {
	var diags diag.Diagnostics

	encoded, err := json.Marshal(storedValuesOf(setting))
	if err != nil {
		diags.AddError("Error Storing ClickHouse Setting Private State", fmt.Sprintf("%+v\n", err))
		return diags
	}

	return private.SetKey(ctx, storedValuesKey, encoded)
}

func getStoredValues(ctx context.Context, private privateState) (*storedValues, diag.Diagnostics) {
	encoded, diags := private.GetKey(ctx, storedValuesKey)
	if diags.HasError() || encoded == nil {
		return nil, diags
	}

	var stored storedValues
	if err := json.Unmarshal(encoded, &stored); err != nil {
		diags.AddError("Error Reading ClickHouse Setting Private State", fmt.Sprintf("%+v\n", err))
		return nil, diags
	}

	return &stored, diags
}
