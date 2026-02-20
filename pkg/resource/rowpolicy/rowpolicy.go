package rowpolicy

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed rowpolicy.md
var rowPolicyDescription string

var (
	_ resource.Resource              = &Resource{}
	_ resource.ResourceWithConfigure = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

// listToStringSlice converts a types.List to []string
func listToStringSlice(ctx context.Context, tfList types.List) ([]string, error) {
	if tfList.IsNull() || tfList.IsUnknown() {
		return []string{}, nil
	}

	var result []string
	diags := tfList.ElementsAs(ctx, &result, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert list to string slice")
	}
	return result, nil
}

// stringSliceToList converts []string to types.List
func stringSliceToList(ctx context.Context, strings []string) (types.List, error) {
	if len(strings) == 0 {
		return types.ListNull(types.StringType), nil
	}

	elements := make([]attr.Value, len(strings))
	for i, s := range strings {
		elements[i] = types.StringValue(s)
	}

	listVal, diags := types.ListValue(types.StringType, elements)
	if diags.HasError() {
		return types.ListNull(types.StringType), fmt.Errorf("failed to convert string slice to list")
	}
	return listVal, nil
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
				Description: "The database of the table to apply the row policy to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"table_name": schema.StringAttribute{
				Required:    true,
				Description: "The table to apply the row policy to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"for_operations": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of operations the row policy applies to (e.g. 'SELECT'). If not specified, defaults to SELECT. Currently only SELECT is supported; this field is designed to support INSERT, UPDATE, DELETE in future ClickHouse versions.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("SELECT"),
					),
				},
			},
			"select_filter": schema.StringAttribute{
				Required:    true,
				Description: "The filter expression used in the USING clause. For example: `tenant_id = 'abc'`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"is_restrictive": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, the policy is restrictive (AND logic). If false (default), the policy is permissive (OR logic).",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"grantee_user_names": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of user names to apply the row policy to.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(
						path.MatchRoot("grantee_role_names"),
						path.MatchRoot("grantee_all"),
						path.MatchRoot("grantee_all_except"),
					),
				},
			},
			"grantee_role_names": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of role names to apply the row policy to.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(
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
			"grantee_all_except": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Apply the row policy to all users and roles except those listed.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(
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

	if r.client != nil {
		isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Checking if service is using replicated storage",
				fmt.Sprintf("%+v\n", err),
			)
			return
		}

		if isReplicatedStorage && !config.ClusterName.IsNull() {
			resp.Diagnostics.AddWarning(
				"Invalid configuration",
				"Your ClickHouse cluster is using Replicated storage for grants, please remove the 'cluster_name' attribute from your RowPolicy resource definition if you encounter any errors.",
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

	userNames, err := listToStringSlice(ctx, plan.GranteeUserNames)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_user_names", err.Error())
		return
	}

	roleNames, err := listToStringSlice(ctx, plan.GranteeRoleNames)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_role_names", err.Error())
		return
	}

	allExcept, err := listToStringSlice(ctx, plan.GranteeAllExcept)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_all_except", err.Error())
		return
	}

	forOperations, err := listToStringSlice(ctx, plan.ForOperations)
	if err != nil {
		resp.Diagnostics.AddError("Invalid for_operations", err.Error())
		return
	}

	rp := dbops.RowPolicy{
		Name:             plan.Name.ValueString(),
		Database:         plan.Database.ValueString(),
		Table:            plan.Table.ValueString(),
		ForOperations:    forOperations,
		SelectFilter:     plan.SelectFilter.ValueString(),
		IsRestrictive:    plan.IsRestrictive.ValueBool(),
		GranteeUserNames: userNames,
		GranteeRoleNames: roleNames,
		GranteeAll:       plan.GranteeAll.ValueBool(),
		GranteeAllExcept: allExcept,
	}

	created, err := r.client.CreateRowPolicy(ctx, rp, plan.ClusterName.ValueStringPointer())
	if err != nil {
		// If the row policy already exists, try to read it instead
		// This allows terraform apply to be idempotent
		if strings.Contains(err.Error(), "already exists") {
			created, err = r.client.GetRowPolicy(ctx, &rp, plan.ClusterName.ValueStringPointer())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Reading ClickHouse Row Policy",
					"Row policy already exists but could not be read: "+err.Error(),
				)
				return
			}
		} else {
			resp.Diagnostics.AddError(
				"Error Creating ClickHouse Row Policy",
				"Could not create row policy, unexpected error: "+err.Error(),
			)
			return
		}
	}

	if created == nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Row Policy",
			"The row policy was created but could not be found in system.row_policies.",
		)
		return
	}

	userNamesList, err := stringSliceToList(ctx, created.GranteeUserNames)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert user names to list", err.Error())
		return
	}

	roleNamesList, err := stringSliceToList(ctx, created.GranteeRoleNames)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert role names to list", err.Error())
		return
	}

	allExceptList, err := stringSliceToList(ctx, created.GranteeAllExcept)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert all except to list", err.Error())
		return
	}

	forOperationsList, err := stringSliceToList(ctx, created.ForOperations)
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert for_operations to list", err.Error())
		return
	}

	// Preserve grantee fields from plan - only set if they were explicitly specified
	granteeAll := types.BoolNull()
	if !plan.GranteeAll.IsNull() {
		granteeAll = types.BoolValue(created.GranteeAll)
	}

	state := RowPolicy{
		ClusterName:      plan.ClusterName,
		Name:             types.StringValue(created.Name),
		Database:         types.StringValue(created.Database),
		Table:            types.StringValue(created.Table),
		ForOperations:    forOperationsList,
		SelectFilter:     types.StringValue(created.SelectFilter),
		IsRestrictive:    types.BoolValue(created.IsRestrictive),
		GranteeUserNames: userNamesList,
		GranteeRoleNames: roleNamesList,
		GranteeAll:       granteeAll,
		GranteeAllExcept: allExceptList,
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

	userNames, err := listToStringSlice(ctx, state.GranteeUserNames)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_user_names", err.Error())
		return
	}

	roleNames, err := listToStringSlice(ctx, state.GranteeRoleNames)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_role_names", err.Error())
		return
	}

	allExcept, err := listToStringSlice(ctx, state.GranteeAllExcept)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_all_except", err.Error())
		return
	}

	forOperations, err := listToStringSlice(ctx, state.ForOperations)
	if err != nil {
		resp.Diagnostics.AddError("Invalid for_operations", err.Error())
		return
	}

	rp := dbops.RowPolicy{
		Name:             state.Name.ValueString(),
		Database:         state.Database.ValueString(),
		Table:            state.Table.ValueString(),
		ForOperations:    forOperations,
		SelectFilter:     state.SelectFilter.ValueString(),
		IsRestrictive:    state.IsRestrictive.ValueBool(),
		GranteeUserNames: userNames,
		GranteeRoleNames: roleNames,
		GranteeAll:       state.GranteeAll.ValueBool(),
		GranteeAllExcept: allExcept,
	}

	result, err := r.client.GetRowPolicy(ctx, &rp, state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Row Policy",
			"Could not read row policy, unexpected error: "+err.Error(),
		)
		return
	}

	if result != nil {
		userNamesList, err := stringSliceToList(ctx, result.GranteeUserNames)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert user names to list", err.Error())
			return
		}

		roleNamesList, err := stringSliceToList(ctx, result.GranteeRoleNames)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert role names to list", err.Error())
			return
		}

		allExceptList, err := stringSliceToList(ctx, result.GranteeAllExcept)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert all except to list", err.Error())
			return
		}

		forOperationsList, err := stringSliceToList(ctx, result.ForOperations)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert for_operations to list", err.Error())
			return
		}

		state.Name = types.StringValue(result.Name)
		state.Database = types.StringValue(result.Database)
		state.Table = types.StringValue(result.Table)
		state.ForOperations = forOperationsList
		// Preserve SelectFilter and IsRestrictive from state since they can't be reliably read from system table
		// state.SelectFilter and state.IsRestrictive already have values from the state
		state.GranteeUserNames = userNamesList
		state.GranteeRoleNames = roleNamesList
		// Preserve GranteeAll from state - only set if it was explicitly specified in the original plan
		if !state.GranteeAll.IsNull() {
			state.GranteeAll = types.BoolValue(result.GranteeAll)
		}
		state.GranteeAllExcept = allExceptList

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.State.RemoveResource(ctx)
	}
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

	userNames, err := listToStringSlice(ctx, plan.GranteeUserNames)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_user_names", err.Error())
		return
	}

	roleNames, err := listToStringSlice(ctx, plan.GranteeRoleNames)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_role_names", err.Error())
		return
	}

	allExcept, err := listToStringSlice(ctx, plan.GranteeAllExcept)
	if err != nil {
		resp.Diagnostics.AddError("Invalid grantee_all_except", err.Error())
		return
	}

	forOperations, err := listToStringSlice(ctx, plan.ForOperations)
	if err != nil {
		resp.Diagnostics.AddError("Invalid for_operations", err.Error())
		return
	}

	rp := dbops.RowPolicy{
		Name:             plan.Name.ValueString(),
		Database:         plan.Database.ValueString(),
		Table:            plan.Table.ValueString(),
		ForOperations:    forOperations,
		SelectFilter:     plan.SelectFilter.ValueString(),
		IsRestrictive:    plan.IsRestrictive.ValueBool(),
		GranteeUserNames: userNames,
		GranteeRoleNames: roleNames,
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
		userNamesList, err := stringSliceToList(ctx, updated.GranteeUserNames)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert user names to list", err.Error())
			return
		}

		roleNamesList, err := stringSliceToList(ctx, updated.GranteeRoleNames)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert role names to list", err.Error())
			return
		}

		allExceptList, err := stringSliceToList(ctx, updated.GranteeAllExcept)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert all except to list", err.Error())
			return
		}

		forOperationsList, err := stringSliceToList(ctx, updated.ForOperations)
		if err != nil {
			resp.Diagnostics.AddError("Failed to convert for_operations to list", err.Error())
			return
		}

		state.Name = types.StringValue(updated.Name)
		state.Database = types.StringValue(updated.Database)
		state.Table = types.StringValue(updated.Table)
		state.ForOperations = forOperationsList
		state.SelectFilter = types.StringValue(updated.SelectFilter)
		state.IsRestrictive = types.BoolValue(updated.IsRestrictive)
		state.GranteeUserNames = userNamesList
		state.GranteeRoleNames = roleNamesList
		state.GranteeAll = types.BoolValue(updated.GranteeAll)
		state.GranteeAllExcept = allExceptList

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
