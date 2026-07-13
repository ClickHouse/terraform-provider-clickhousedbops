package maskingpolicy

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

// maskFingerprintKey names the private-state entry holding the hash of the server's UPDATE clause at the last write.
const maskFingerprintKey = "mask_fingerprint"

//go:embed maskingpolicy.md
var maskingPolicyDescription string

// nonBlank rejects empty and whitespace-only strings, which whereOrNil would otherwise drop silently.
var nonBlank = regexp.MustCompile(`\S`)

var (
	_ resource.Resource                     = &Resource{}
	_ resource.ResourceWithConfigure        = &Resource{}
	_ resource.ResourceWithConfigValidators = &Resource{}
	_ resource.ResourceWithModifyPlan       = &Resource{}
	_ resource.ResourceWithImportState      = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_masking_policy"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The system-assigned ID for the masking policy.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the masking policy.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"database_name": schema.StringAttribute{
				Required:    true,
				Description: "The database of the table the masking policy applies to. Must be a concrete name; wildcards (`*`) are not supported.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"table_name": schema.StringAttribute{
				Required:    true,
				Description: "The table the masking policy applies to. Must be a concrete name; wildcards (`*`) are not supported.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"masks": schema.MapAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "Map of column name to the ClickHouse expression that replaces it (the `UPDATE column = expression` clause). The expression is interpolated verbatim, e.g. `'** redacted **'` or `concat(splitByChar('.', clientIp)[1], '.x.x')`.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(nonBlank, "must not be blank")),
					mapvalidator.ValueStringsAre(stringvalidator.RegexMatches(nonBlank, "must not be blank")),
				},
			},
			"where_expression": schema.StringAttribute{
				Optional:    true,
				Description: "Optional `WHERE` condition; the columns are only masked for rows matching it. For example `ownerId NOT IN ('team_a', 'team_b')`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(nonBlank, "must not be blank"),
				},
			},
			"grantee_names": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Set of user or role names the masking policy applies to. ClickHouse resolves each name to a user before a role, so users and roles are not distinguished here.",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
			},
			"grantee_all_except": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Apply the masking policy to all users and roles, excluding those listed. An empty set applies to everyone with no exclusions.",
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
				Description: "Optional priority. When several policies touch the same column, they are applied from highest to lowest priority. Must be non-negative. Defaults to 0.",
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
		},
		MarkdownDescription: maskingPolicyDescription,
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
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var stateWhere, planWhere types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("where_expression"), &stateWhere)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("where_expression"), &planWhere)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateHasWhere := !stateWhere.IsNull() && stateWhere.ValueString() != ""
	planClearsWhere := !planWhere.IsUnknown() && (planWhere.IsNull() || planWhere.ValueString() == "")
	if stateHasWhere && planClearsWhere {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("where_expression"))
	}
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MaskingPolicy
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mp, diags := plan.toDBOps(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateMaskingPolicy(ctx, mp)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			resp.Diagnostics.AddError(
				"ClickHouse Masking Policy already exists",
				fmt.Sprintf("A masking policy %q already exists on %s.%s. Import it with `terraform import <resource> %s.%s.%s` instead of recreating it.", mp.Name, mp.Database, mp.Table, mp.Database, mp.Table, mp.Name),
			)
			return
		}
		resp.Diagnostics.AddError("Error Creating ClickHouse Masking Policy", "Could not create masking policy, unexpected error: "+err.Error())
		return
	}

	var state MaskingPolicy
	resp.Diagnostics.Append(state.fromDBOps(created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Masks = plan.Masks
	state.WhereExpression = plan.WhereExpression
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, maskFingerprintKey, []byte(created.AssignmentsHash))...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MaskingPolicy
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetMaskingPolicyByID(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Reading ClickHouse Masking Policy", "Could not read masking policy, unexpected error: "+err.Error())
		return
	}

	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	stateWhere := state.WhereExpression

	resp.Diagnostics.Append(state.fromDBOps(result)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.normalizedEquals(ctx, stateWhere.ValueString(), result.Where) {
		state.WhereExpression = stateWhere
	}

	// If mask key fingerprint changed, we need to clean masks in state to produce diff during plan.
	fingerprint, d := req.Private.GetKey(ctx, maskFingerprintKey)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	switch {
	case fingerprint == nil:
		resp.Diagnostics.Append(resp.Private.SetKey(ctx, maskFingerprintKey, []byte(result.AssignmentsHash))...)
	case string(fingerprint) != result.AssignmentsHash:
		state.Masks = types.MapNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MaskingPolicy
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mp, diags := plan.toDBOps(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// id is computed and marked unknown in the plan on update, so key the ALTER on the prior state's id.
	mp.ID = state.ID.ValueString()

	updated, err := r.client.UpdateMaskingPolicy(ctx, mp)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating ClickHouse Masking Policy", "Could not update masking policy, unexpected error: "+err.Error())
		return
	}

	var newState MaskingPolicy
	resp.Diagnostics.Append(newState.fromDBOps(updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newState.Masks = plan.Masks
	newState.WhereExpression = plan.WhereExpression
	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, maskFingerprintKey, []byte(updated.AssignmentsHash))...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MaskingPolicy
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMaskingPolicy(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting ClickHouse Masking Policy", "Could not delete masking policy, unexpected error: "+err.Error())
		return
	}
}

// ImportState imports a masking policy identified either by its UUID or "database.table.name".
func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var result *dbops.MaskingPolicy
	var err error
	if _, parseErr := uuid.Parse(req.ID); parseErr == nil {
		result, err = r.client.GetMaskingPolicyByID(ctx, req.ID)
	} else {
		parts := strings.SplitN(req.ID, ".", 3)
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			resp.Diagnostics.AddError(
				"Invalid import ID",
				fmt.Sprintf("Expected import ID as a masking policy UUID or in the form \"database.table.name\", got %q", req.ID),
			)
			return
		}
		result, err = r.client.GetMaskingPolicy(ctx, &dbops.MaskingPolicy{Database: parts[0], Table: parts[1], Name: parts[2]})
	}
	if err != nil {
		resp.Diagnostics.AddError("Error Reading ClickHouse Masking Policy", "Could not read masking policy, unexpected error: "+err.Error())
		return
	}
	if result == nil {
		resp.Diagnostics.AddError("Masking Policy Not Found", fmt.Sprintf("Masking policy %q not found", req.ID))
		return
	}

	var state MaskingPolicy
	resp.Diagnostics.Append(state.fromDBOps(result)...)
	state.Masks = types.MapNull(types.StringType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(resp.Private.SetKey(ctx, maskFingerprintKey, []byte(result.AssignmentsHash))...)
}

// normalizedEquals reports whether an expression returned by the server is equal to state.
func (r *Resource) normalizedEquals(ctx context.Context, state, server string) bool {
	if server == state {
		return true
	}
	if state == "" {
		return false
	}

	normalized, err := r.client.NormalizeExpression(ctx, state)
	if err != nil {
		return true
	}
	return server == normalized
}
