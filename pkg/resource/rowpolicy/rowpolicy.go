package rowpolicy

import (
	"context"
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithConfigure   = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

// setToStringSlice converts a types.Set to []string
func setToStringSlice(ctx context.Context, tfSet types.Set) ([]string, diag.Diagnostics) {
	if tfSet.IsNull() || tfSet.IsUnknown() {
		return []string{}, nil
	}

	var result []string
	diags := tfSet.ElementsAs(ctx, &result, false)
	if diags.HasError() {
		return nil, diags
	}
	return result, diags
}

// stringSliceToSet converts []string to types.Set
func stringSliceToSet(ctx context.Context, strings []string) (types.Set, diag.Diagnostics) {
	if len(strings) == 0 {
		return types.SetNull(types.StringType), nil
	}

	elements := make([]attr.Value, len(strings))
	for i, s := range strings {
		elements[i] = types.StringValue(s)
	}

	setVal, diags := types.SetValue(types.StringType, elements)
	if diags.HasError() {
		return types.SetNull(types.StringType), diags
	}
	return setVal, diags
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the row policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
			"grantee_user_names": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Set of user names to apply the row policy to.",
				Validators: []validator.Set{
					setvalidator.ConflictsWith(
						path.MatchRoot("grantee_role_names"),
						path.MatchRoot("grantee_all"),
						path.MatchRoot("grantee_all_except"),
					),
				},
			},
			"grantee_role_names": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Set of role names to apply the row policy to.",
				Validators: []validator.Set{
					setvalidator.ConflictsWith(
						path.MatchRoot("grantee_user_names"),
						path.MatchRoot("grantee_all"),
						path.MatchRoot("grantee_all_except"),
					),
				},
			},
			"grantee_all": schema.BoolAttribute{
				Optional:    true,
				Description: "Apply the row policy to all users and roles.",
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(
						path.MatchRoot("grantee_user_names"),
						path.MatchRoot("grantee_role_names"),
						path.MatchRoot("grantee_all_except"),
					),
				},
			},
			"grantee_all_except": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Apply the row policy to all users and roles except those listed.",
				Validators: []validator.Set{
					setvalidator.ConflictsWith(
						path.MatchRoot("grantee_user_names"),
						path.MatchRoot("grantee_role_names"),
						path.MatchRoot("grantee_all"),
					),
				},
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

	userNames, diags := setToStringSlice(ctx, plan.GranteeUserNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleNames, diags := setToStringSlice(ctx, plan.GranteeRoleNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allExcept, diags := setToStringSlice(ctx, plan.GranteeAllExcept)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp := dbops.RowPolicy{
		Name:             plan.Name.ValueString(),
		Database:         plan.Database.ValueString(),
		Table:            plan.Table.ValueString(),
		SelectFilter:     plan.SelectFilter.ValueString(),
		IsRestrictive:    plan.IsRestrictive.ValueBool(),
		GranteeNames:     append(slices.Clone(userNames), roleNames...),
		GranteeAll:       plan.GranteeAll.ValueBool(),
		GranteeAllExcept: allExcept,
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

	state := RowPolicy{
		ClusterName:      plan.ClusterName,
		Name:             types.StringValue(created.Name),
		Database:         types.StringValue(created.Database),
		Table:            types.StringValue(created.Table),
		SelectFilter:     plan.SelectFilter,
		IsRestrictive:    types.BoolValue(created.IsRestrictive),
		GranteeUserNames: plan.GranteeUserNames,
		GranteeRoleNames: plan.GranteeRoleNames,
		GranteeAll:       plan.GranteeAll,
		GranteeAllExcept: plan.GranteeAllExcept,
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RowPolicy
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp := dbops.RowPolicy{
		Name:     state.Name.ValueString(),
		Database: state.Database.ValueString(),
		Table:    state.Table.ValueString(),
	}

	result, err := r.client.GetRowPolicy(ctx, &rp, state.ClusterName.ValueStringPointer())
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

	stateUsers, diags := setToStringSlice(ctx, state.GranteeUserNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateRoles, diags := setToStringSlice(ctx, state.GranteeRoleNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// ClickHouse stores row-policy grantees as one untyped list (apply_to_list), so the read can't
	// tell users from roles. Preserve each grantee's configured type for names still present, and
	// treat a name added out of band as a user, which surfaces the drift for the user to reclassify.
	inDB := make(map[string]bool, len(result.GranteeNames))
	for _, n := range result.GranteeNames {
		inDB[n] = true
	}
	known := make(map[string]bool, len(stateUsers)+len(stateRoles))
	userNames := make([]string, 0, len(stateUsers))
	roleNames := make([]string, 0, len(stateRoles))
	for _, u := range stateUsers {
		known[u] = true
		if inDB[u] {
			userNames = append(userNames, u)
		}
	}
	for _, ro := range stateRoles {
		known[ro] = true
		if inDB[ro] {
			roleNames = append(roleNames, ro)
		}
	}
	for _, n := range result.GranteeNames {
		if !known[n] {
			userNames = append(userNames, n)
		}
	}

	userNamesSet, diags := stringSliceToSet(ctx, userNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleNamesSet, diags := stringSliceToSet(ctx, roleNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allExceptSet, diags := stringSliceToSet(ctx, result.GranteeAllExcept)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Name = types.StringValue(result.Name)
	state.Database = types.StringValue(result.Database)
	state.Table = types.StringValue(result.Table)
	// is_restrictive is a plain bool in system.row_policies, so reconcile it to catch out-of-band
	// permissive/restrictive flips.
	state.IsRestrictive = types.BoolValue(result.IsRestrictive)

	// Reconcile select_filter. ClickHouse stores it in a normalized form, so comparing the raw
	// strings would drift against the config on whitespace/parenthesization alone. Normalize the
	// configured filter to the same canonical form and only surface the DB value when it genuinely
	// differs (real drift). On import (no filter in state yet) adopt the DB value directly.
	if state.SelectFilter.IsNull() || state.SelectFilter.ValueString() == "" {
		state.SelectFilter = types.StringValue(result.SelectFilter)
	} else {
		normalized, err := r.client.NormalizeRowPolicyFilter(ctx, state.SelectFilter.ValueString(), state.ClusterName.ValueStringPointer())
		if err == nil && normalized != result.SelectFilter {
			state.SelectFilter = types.StringValue(result.SelectFilter)
		}
	}

	// Reconcile grantees from the DB to surface out-of-band changes. apply_to_all is 1 for both
	// `TO ALL` and `TO ALL EXCEPT …`; only the former maps to grantee_all, so an except list keeps
	// grantee_all null (avoiding the ConflictsWith). false also maps to null (an unset optional).
	state.GranteeUserNames = userNamesSet
	state.GranteeRoleNames = roleNamesSet
	state.GranteeAllExcept = allExceptSet
	if result.GranteeAll && len(result.GranteeAllExcept) == 0 {
		state.GranteeAll = types.BoolValue(true)
	} else {
		state.GranteeAll = types.BoolNull()
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

	userNames, diags := setToStringSlice(ctx, plan.GranteeUserNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleNames, diags := setToStringSlice(ctx, plan.GranteeRoleNames)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	allExcept, diags := setToStringSlice(ctx, plan.GranteeAllExcept)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	rp := dbops.RowPolicy{
		Name:             plan.Name.ValueString(),
		Database:         plan.Database.ValueString(),
		Table:            plan.Table.ValueString(),
		SelectFilter:     plan.SelectFilter.ValueString(),
		IsRestrictive:    plan.IsRestrictive.ValueBool(),
		GranteeNames:     append(slices.Clone(userNames), roleNames...),
		GranteeAll:       plan.GranteeAll.ValueBool(),
		GranteeAllExcept: allExcept,
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
		state.Name = types.StringValue(updated.Name)
		state.Database = types.StringValue(updated.Database)
		state.Table = types.StringValue(updated.Table)
		state.SelectFilter = plan.SelectFilter
		state.IsRestrictive = types.BoolValue(updated.IsRestrictive)
		state.GranteeUserNames = plan.GranteeUserNames
		state.GranteeRoleNames = plan.GranteeRoleNames
		state.GranteeAll = plan.GranteeAll
		state.GranteeAllExcept = plan.GranteeAllExcept

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

	err := r.client.DeleteRowPolicy(ctx, state.Name.ValueString(), state.Database.ValueString(), state.Table.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Row Policy",
			"Could not delete row policy, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports a row policy from an ID of the form "database.table.name".
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ".", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			fmt.Sprintf("Expected import ID in the form \"database.table.name\", got %q", req.ID),
		)
		return
	}

	rp := dbops.RowPolicy{
		Database: parts[0],
		Table:    parts[1],
		Name:     parts[2],
	}

	result, err := r.client.GetRowPolicy(ctx, &rp, nil)
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

	granteeNames, diag := stringSliceToSet(ctx, result.GranteeNames)
	resp.Diagnostics.Append(diag...)

	granteeAllExcept, diag := stringSliceToSet(ctx, result.GranteeAllExcept)
	resp.Diagnostics.Append(diag...)

	state := RowPolicy{
		Database:      types.StringValue(rp.Database),
		Table:         types.StringValue(rp.Table),
		Name:          types.StringValue(rp.Name),
		SelectFilter:  types.StringValue(result.SelectFilter),
		IsRestrictive: types.BoolValue(result.IsRestrictive),
		// ClickHouse returns union of users and roles, so we can't distinguish it here
		GranteeUserNames: granteeNames,
		GranteeAll:       types.BoolValue(result.GranteeAll),
		GranteeAllExcept: granteeAllExcept,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
