package maskingpolicy

import (
	"context"
	_ "embed"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
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

//go:embed maskingpolicy.md
var maskingPolicyDescription string

var (
	_ resource.Resource              = &Resource{}
	_ resource.ResourceWithConfigure = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

func listToStringSlice(ctx context.Context, l types.List) ([]string, error) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	var out []string
	if diags := l.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil, fmt.Errorf("failed to convert list to string slice")
	}
	return out, nil
}

func mapToColumnMasks(ctx context.Context, m types.Map) ([]dbops.ColumnMask, error) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	var raw map[string]string
	if diags := m.ElementsAs(ctx, &raw, false); diags.HasError() {
		return nil, fmt.Errorf("failed to convert masks map")
	}
	masks := make([]dbops.ColumnMask, 0, len(raw))
	for col, expr := range raw {
		masks = append(masks, dbops.ColumnMask{Column: col, Expression: expr})
	}
	sort.Slice(masks, func(i, j int) bool { return masks[i].Column < masks[j].Column })
	return masks, nil
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_masking_policy"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the masking policy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"database_name": schema.StringAttribute{
				Required:    true,
				Description: "The database of the table the masking policy applies to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"table_name": schema.StringAttribute{
				Required:    true,
				Description: "The table the masking policy applies to.",
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
				},
			},
			"where_expression": schema.StringAttribute{
				Optional:    true,
				Description: "Optional `WHERE` condition; the columns are only masked for rows matching it. For example `ownerId NOT IN ('team_a', 'team_b')`.",
			},
			"grantee_user_names": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of user names the masking policy applies to.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(
						path.MatchRoot("grantee_all"),
						path.MatchRoot("grantee_all_except"),
					),
				},
			},
			"grantee_role_names": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "List of role names the masking policy applies to.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(
						path.MatchRoot("grantee_all"),
						path.MatchRoot("grantee_all_except"),
					),
				},
			},
			"grantee_all": schema.BoolAttribute{
				Optional:    true,
				Description: "Apply the masking policy to all users and roles.",
			},
			"grantee_all_except": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Apply the masking policy to all users and roles except those listed.",
				Validators: []validator.List{
					listvalidator.ConflictsWith(
						path.MatchRoot("grantee_user_names"),
						path.MatchRoot("grantee_role_names"),
					),
				},
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Description: "Optional priority. When several policies touch the same column, they are applied from highest to lowest priority. Defaults to 0 in ClickHouse.",
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

func (r *Resource) modelToDBOps(ctx context.Context, m MaskingPolicy) (dbops.MaskingPolicy, error) {
	masks, err := mapToColumnMasks(ctx, m.Masks)
	if err != nil {
		return dbops.MaskingPolicy{}, err
	}
	users, err := listToStringSlice(ctx, m.GranteeUserNames)
	if err != nil {
		return dbops.MaskingPolicy{}, err
	}
	roles, err := listToStringSlice(ctx, m.GranteeRoleNames)
	if err != nil {
		return dbops.MaskingPolicy{}, err
	}
	allExcept, err := listToStringSlice(ctx, m.GranteeAllExcept)
	if err != nil {
		return dbops.MaskingPolicy{}, err
	}

	return dbops.MaskingPolicy{
		Name:             m.Name.ValueString(),
		Database:         m.Database.ValueString(),
		Table:            m.Table.ValueString(),
		Masks:            masks,
		Where:            m.WhereExpression.ValueString(),
		GranteeUserNames: users,
		GranteeRoleNames: roles,
		GranteeAll:       m.GranteeAll.ValueBool(),
		GranteeAllExcept: allExcept,
		Priority:         m.Priority.ValueInt64Pointer(),
	}, nil
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MaskingPolicy
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mp, err := r.modelToDBOps(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid masking policy configuration", err.Error())
		return
	}

	created, err := r.client.CreateMaskingPolicy(ctx, mp)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating ClickHouse Masking Policy", "Could not create masking policy, unexpected error: "+err.Error())
		return
	}
	if created == nil {
		resp.Diagnostics.AddError("Error Creating ClickHouse Masking Policy", "The masking policy was created but could not be found via SHOW MASKING POLICIES.")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MaskingPolicy
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mp, err := r.modelToDBOps(ctx, state)
	if err != nil {
		resp.Diagnostics.AddError("Invalid masking policy state", err.Error())
		return
	}

	found, err := r.client.GetMaskingPolicy(ctx, &mp)
	if err != nil {
		resp.Diagnostics.AddError("Error Reading ClickHouse Masking Policy", "Could not read masking policy, unexpected error: "+err.Error())
		return
	}

	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Definition fields are authoritative from state; only existence is verified on read.
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan MaskingPolicy
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mp, err := r.modelToDBOps(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Invalid masking policy configuration", err.Error())
		return
	}

	if _, err := r.client.UpdateMaskingPolicy(ctx, mp); err != nil {
		resp.Diagnostics.AddError("Error Updating ClickHouse Masking Policy", "Could not update masking policy, unexpected error: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MaskingPolicy
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMaskingPolicy(ctx, state.Name.ValueString(), state.Database.ValueString(), state.Table.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error Deleting ClickHouse Masking Policy", "Could not delete masking policy, unexpected error: "+err.Error())
		return
	}
}
