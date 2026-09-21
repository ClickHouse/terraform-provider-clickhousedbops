package revokeprivilege

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
)

//go:embed revokeprivilege.md
var revokePrivilegeDescription string

var (
	_ resource.Resource                   = &Resource{}
	_ resource.ResourceWithConfigure      = &Resource{}
	_ resource.ResourceWithModifyPlan     = &Resource{}
	_ resource.ResourceWithValidateConfig = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_revoke_privilege"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	validPrivileges := make([]string, 0)
	catalog := grants.Parsed()
	for privilege := range catalog.Scopes {
		validPrivileges = append(validPrivileges, privilege)
	}
	for alias := range catalog.Aliases {
		validPrivileges = append(validPrivileges, alias)
	}
	for groupName := range catalog.Groups {
		validPrivileges = append(validPrivileges, groupName)
	}

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the cluster on which to create the partial revoke. Leave null for ClickHouse Cloud and replicated access storage.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"privilege_name": schema.StringAttribute{
				Required:    true,
				Description: "The privilege to partially revoke, such as `SELECT`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(validPrivileges...),
				},
			},
			"database_name": schema.StringAttribute{
				Optional:    true,
				Description: "The database at which to revoke the privilege. Null represents all databases.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.NoneOf("*"),
				},
			},
			"table_name": schema.StringAttribute{
				Optional:    true,
				Description: "The table at which to revoke the privilege. Null represents all tables.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.NoneOf("*"),
					stringvalidator.AlsoRequires(path.MatchRoot("database_name")),
				},
			},
			"column_name": schema.StringAttribute{
				Optional:    true,
				Description: "The column at which to revoke the privilege.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.AlsoRequires(
						path.MatchRoot("database_name"),
						path.MatchRoot("table_name"),
					),
				},
			},
			"access_object": schema.StringAttribute{
				Optional:    true,
				Description: "The access object at which to revoke a USER_NAME- or DEFINER-scoped privilege. Supports a trailing `*` prefix pattern.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.NoneOf("*"),
					stringvalidator.ConflictsWith(
						path.MatchRoot("database_name"),
						path.MatchRoot("table_name"),
						path.MatchRoot("column_name"),
					),
				},
			},
			"grantee_user_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the user from which to partially revoke the privilege.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("grantee_role_name")),
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("grantee_user_name"),
						path.MatchRoot("grantee_role_name"),
					),
				},
			},
			"grantee_role_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the role from which to partially revoke the privilege.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("grantee_user_name")),
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("grantee_user_name"),
						path.MatchRoot("grantee_role_name"),
					),
				},
			},
			"grant_option_only": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "When true, revoke only the ability to grant the privilege to others (`REVOKE GRANT OPTION FOR`) while retaining the privilege itself.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
		MarkdownDescription: revokePrivilegeDescription,
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData != nil {
		r.client = req.ProviderData.(dbops.Client)
	}
}

func validateScope(config RevokePrivilege, diags *diag.Diagnostics) {
	if config.Privilege.IsUnknown() {
		return
	}

	catalog := grants.Parsed()
	privilege := config.Privilege.ValueString()
	if alias := catalog.Aliases[privilege]; alias != "" {
		diags.AddAttributeError(
			path.Root("privilege_name"),
			"Cannot use alias",
			fmt.Sprintf("%q is an alias for %q. Please use %q instead", privilege, alias, alias),
		)
		return
	}

	// ClickHouse can expand a group into several scope-dependent child rows.
	// Treating any one child as the resource would make read and delete
	// incomplete. A future authoritative grant-set resource can expand groups
	// into explicit leaf PartialRevoke values before batching them.
	if len(catalog.Groups[privilege]) != 0 {
		diags.AddAttributeError(
			path.Root("privilege_name"),
			"Privilege Groups Are Not Supported",
			fmt.Sprintf("%q is a privilege group. Declare a separate partial revoke for each canonical leaf privilege that should be removed.", privilege),
		)
		return
	}

	attrs, _, ok := grants.ScopeAttributesFor(privilege)
	if !ok {
		diags.AddAttributeError(
			path.Root("privilege_name"),
			"Unsupported Privilege",
			fmt.Sprintf("%q privilege_name is currently unsupported", privilege),
		)
		return
	}

	// ClickHouse does not support subtracting a narrower source or source
	// pattern from a source grant. It must be replaced with a new source grant.
	if grants.ScopeFor(privilege) == "SOURCE" {
		diags.AddAttributeError(
			path.Root("privilege_name"),
			"Source Partial Revokes Are Not Supported",
			"ClickHouse does not support partial revokes for READ or WRITE source grants. Revoke the complete source grant and create the desired replacement instead.",
		)
		return
	}

	checkAttr := func(attrName string, isSupported, isSet bool) {
		if !isSet || isSupported {
			return
		}
		diags.AddAttributeError(
			path.Root(attrName),
			"Invalid Partial Privilege Revoke",
			fmt.Sprintf("%q must be null when 'privilege_name' is %q", attrName, privilege),
		)
	}

	checkAttr("database_name", attrs.Database, !config.Database.IsNull())
	checkAttr("table_name", attrs.Table, !config.Table.IsNull())
	checkAttr("column_name", attrs.Column, !config.Column.IsNull())
	checkAttr("access_object", attrs.AccessObject, !config.AccessObject.IsNull())
}

func (r *Resource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config RevokePrivilege
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if !resp.Diagnostics.HasError() {
		validateScope(config, &resp.Diagnostics)
	}
}

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var config RevokePrivilege
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateScope(config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || r.client == nil || config.ClusterName.IsNull() {
		return
	}

	isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Could not check if service is using replicated storage",
			fmt.Sprintf("Skipping validation. If you are using replicated storage, remove 'cluster_name'. Error: %+v", err),
		)
		return
	}
	if isReplicatedStorage {
		resp.Diagnostics.AddWarning(
			"Invalid configuration",
			"Your ClickHouse cluster uses replicated access storage. Remove 'cluster_name' if the partial revoke fails.",
		)
	}
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RevokePrivilege
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	partialRevoke := plan.toPartialRevoke()
	created, err := r.client.CreatePartialRevoke(ctx, partialRevoke, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Partial Privilege Revoke",
			"Could not create partial privilege revoke: "+err.Error(),
		)
		return
	}
	if created == nil {
		existing, lookupErr := r.client.GetAllPartialRevokesForGrantee(
			ctx,
			partialRevoke.GranteeUserName,
			partialRevoke.GranteeRoleName,
			plan.ClusterName.ValueStringPointer(),
		)
		if lookupErr != nil {
			resp.Diagnostics.AddError(
				"Error Checking Partial Privilege Revokes",
				"Could not check for an overlapping partial revoke: "+lookupErr.Error(),
			)
			return
		}
		for idx := range existing {
			if dbops.CoversPartialRevoke(existing[idx], partialRevoke) {
				resp.Diagnostics.AddError(
					"Overlapping Partial Privilege Revoke",
					fmt.Sprintf("The requested partial revoke is already covered by an existing partial revoke (%s). Manage overlapping negative rights in one resource hierarchy to avoid ambiguous ownership.", dbops.DescribePartialRevoke(existing[idx])),
				)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Partial Privilege Revoke Was Not Created",
			"ClickHouse accepted the REVOKE statement but did not create a partial-revoke row. A partial revoke must narrow an existing broader grant for the same grantee.",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, toState(*created, plan.ClusterName))...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RevokePrivilege
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.client.GetPartialRevoke(ctx, new(state.toPartialRevoke()), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Partial Privilege Revoke",
			"Could not read partial privilege revoke: "+err.Error(),
		)
		return
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, toState(*found, state.ClusterName))...)
}

func (r *Resource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	panic("Update of revoke privilege resource is not supported")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RevokePrivilege
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePartialRevoke(ctx, state.toPartialRevoke(), state.ClusterName.ValueStringPointer()); err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Partial Privilege Revoke",
			"Could not delete partial privilege revoke: "+err.Error(),
		)
	}
}
