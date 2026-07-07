package rowpolicy

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed rowpolicy.md
var rowPolicyDescription string

var (
	_ resource.Resource                     = &Resource{}
	_ resource.ResourceWithConfigure        = &Resource{}
	_ resource.ResourceWithImportState      = &Resource{}
	_ resource.ResourceWithConfigValidators = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_row_policy"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the cluster to create the resource into. If omitted, resource will be created on the replica hit by the query.\nThis field must be left null when using a ClickHouse Cloud cluster.\n",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The system-assigned ID for the row policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the row policy.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"database_name": schema.StringAttribute{
				Required:    true,
				Description: "The database of the table to apply the row policy to. Must be a concrete name; wildcards (`*`) are not supported.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"table_name": schema.StringAttribute{
				Required:    true,
				Description: "The table to apply the row policy to. Must be a concrete name; wildcards (`*`) are not supported.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"select_filter": schema.StringAttribute{
				Required:    true,
				Description: "The filter expression used in the USING clause. For example: `tenant_id = 'abc'`.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_restrictive": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the policy is restrictive (AND logic). If false (default), the policy is permissive (OR logic).",
			},
			"grantee_names": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Set of user or role names the row policy applies to. ClickHouse stores these as one untyped grantee list and resolves each name to a user before a role, so users and roles are not distinguished here.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"grantee_all_except": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Apply the row policy to all users and roles, excluding those listed. An empty set applies to everyone with no exclusions.",
			},
		},
		MarkdownDescription: rowPolicyDescription,
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(dbops.Client)
}

func (r *Resource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("grantee_names"),
			path.MatchRoot("grantee_all_except"),
		),
	}
}

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config RowPolicy
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only check replicated storage when cluster_name is set, to avoid an unnecessary
	// connection during plan (e.g. terraform plan -refresh=false with no reachable server).
	if r.client != nil && !config.ClusterName.IsNull() {
		isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Could not check if service is using replicated storage",
				fmt.Sprintf("Skipping validation. If you are using replicated storage, please remove the 'cluster_name' attribute from your resource definition. Error: %+v", err),
			)
			return
		}

		if isReplicatedStorage {
			resp.Diagnostics.AddWarning(
				"Invalid configuration",
				"Your ClickHouse cluster is using Replicated storage for access objects, please remove the 'cluster_name' attribute from your RowPolicy resource definition if you encounter any errors.",
			)
		}
	}
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RowPolicy
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp, diags := plan.toDBOps(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateRowPolicy(ctx, rp, plan.ClusterName.ValueStringPointer())
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			resp.Diagnostics.AddError(
				"ClickHouse Row Policy already exists",
				fmt.Sprintf("A row policy %q already exists on %s.%s. Import it with `terraform import <resource> %s.%s.%s` instead of recreating it.", rp.Name, rp.Database, rp.Table, rp.Database, rp.Table, rp.Name),
			)
			return
		}

		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Row Policy",
			"Could not create row policy, unexpected error: "+err.Error(),
		)
		return
	}

	if created == nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Row Policy",
			"The row policy was created but could not be found in system.row_policies.",
		)
		return
	}

	var state RowPolicy
	resp.Diagnostics.Append(state.fromDBOps(created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.ClusterName = plan.ClusterName
	state.SelectFilter = plan.SelectFilter // store non-normalized version to avoid diff
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RowPolicy
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetRowPolicyByID(ctx, state.ID.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Row Policy",
			"Could not read row policy, unexpected error: "+err.Error(),
		)
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.fromDBOps(result)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Compare normalized select_filters to avoid unnecessary diffs
	if state.SelectFilter.IsNull() || state.SelectFilter.ValueString() == "" {
		state.SelectFilter = types.StringValue(result.SelectFilter)
	} else {
		normalized, err := r.client.NormalizeRowPolicyFilter(ctx, state.SelectFilter.ValueString(), state.ClusterName.ValueStringPointer())
		if err == nil && normalized != result.SelectFilter {
			state.SelectFilter = types.StringValue(result.SelectFilter)
		}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state RowPolicy
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

	rp, diags := plan.toDBOps(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.client.UpdateRowPolicy(ctx, rp, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating ClickHouse Row Policy",
			"Could not update row policy, unexpected error: "+err.Error(),
		)
		return
	}

	if updated != nil {
		resp.Diagnostics.Append(state.fromDBOps(updated)...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.SelectFilter = plan.SelectFilter

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.Diagnostics.AddError(
			"Error Updating ClickHouse Row Policy",
			"The row policy was updated but could not be found in system.row_policies.",
		)
	}
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RowPolicy
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRowPolicy(ctx, state.ID.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Row Policy",
			"Could not delete row policy, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports a row policy identified either by its UUID or by an ID of the form "database.table.name".
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var result *dbops.RowPolicy
	var err error
	if _, parseErr := uuid.Parse(req.ID); parseErr == nil {
		result, err = r.client.GetRowPolicyByID(ctx, req.ID, nil)
	} else {
		parts := strings.SplitN(req.ID, ".", 3)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			resp.Diagnostics.AddError(
				"Invalid import ID",
				fmt.Sprintf("Expected import ID as a row policy UUID or in the form \"database.table.name\", got %q", req.ID),
			)
			return
		}
		result, err = r.client.GetRowPolicy(ctx, &dbops.RowPolicy{Database: parts[0], Table: parts[1], Name: parts[2]}, nil)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Row Policy",
			"Could not read row policy, unexpected error: "+err.Error(),
		)
		return
	}
	if result == nil {
		resp.Diagnostics.AddError("Row Policy Not Found", fmt.Sprintf("Row policy %q not found", req.ID))
		return
	}

	// On import there is no configured filter to reconcile against, so adopt the stored value.
	var state RowPolicy
	resp.Diagnostics.Append(state.fromDBOps(result)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
