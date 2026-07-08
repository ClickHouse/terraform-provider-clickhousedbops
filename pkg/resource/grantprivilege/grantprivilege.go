package grantprivilege

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

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

//go:embed grantprivilege.md
var grantPrivilegeDescription string

var (
	_ resource.Resource                   = &Resource{}
	_ resource.ResourceWithConfigure      = &Resource{}
	_ resource.ResourceWithValidateConfig = &Resource{}
)

func NewResource() resource.Resource {
	return &Resource{}
}

type Resource struct {
	client dbops.Client
}

var _ resource.ResourceWithValidateConfig = &Resource{}

func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_grant_privilege"
}

func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	validPrivileges := make([]string, 0)

	upstrGrts := grants.Parsed()

	for privilege := range upstrGrts.Scopes {
		validPrivileges = append(validPrivileges, privilege)
	}

	for alias := range upstrGrts.Aliases {
		validPrivileges = append(validPrivileges, alias)
	}

	for groupName := range upstrGrts.Groups {
		validPrivileges = append(validPrivileges, groupName)
	}

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the cluster to create the resource into. If omitted, resource will be created on the replica hit by the query.\nThis field must be left null when using a ClickHouse Cloud cluster.\nWhen using a self hosted ClickHouse instance, this field should only be set when there is more than one replica and you are not using 'replicated' storage for user_directory.\n",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"privilege_name": schema.StringAttribute{
				Required:    true,
				Description: "The privilege to grant, such as `CREATE DATABASE`, `SELECT`, etc. See https://clickhouse.com/docs/en/sql-reference/statements/grant#privileges.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(validPrivileges...),
				},
			},
			"database_name": schema.StringAttribute{
				Optional:    true,
				Description: "The name of the database to grant privilege on. Defaults to all databases if left null",
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
				Description: "The name of the table to grant privilege on. Defaults to all tables if left null.",
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
				Description: "The name of the column in `table_name` to grant privilege on.",
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
			"grantee_user_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the `user` to grant privileges to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{path.MatchRoot("grantee_role_name")}...),
					stringvalidator.AtLeastOneOf(path.Expressions{
						path.MatchRoot("grantee_user_name"),
						path.MatchRoot("grantee_role_name"),
					}...),
				},
			},
			"grantee_role_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the `role` to grant privileges to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.Expressions{path.MatchRoot("grantee_user_name")}...),
					stringvalidator.AtLeastOneOf(path.Expressions{
						path.MatchRoot("grantee_user_name"),
						path.MatchRoot("grantee_role_name"),
					}...),
				},
			},
			"grant_option": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, the grantee will be able to grant the same privileges to others.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"current_grants": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "If true, emit `GRANT CURRENT GRANTS(...)` so the privilege is copied from the grantor's own grants instead of granted directly. Required on ClickHouse Cloud for broad privileges (e.g. `ALL`, or `SELECT` on `*.*`) that the admin user holds but cannot transfer directly. Note: the effective grants depend on what the grantor holds at apply time, so drift on a `current_grants` grant is not reconciled.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
		MarkdownDescription: grantPrivilegeDescription,
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(dbops.Client)
}

// validateScope errors when target attributes are set on a privilege whose scope does not support them.
func validateScope(config GrantPrivilege, diags *diag.Diagnostics) {
	if config.Privilege.IsUnknown() {
		return
	}

	upstrGrts := grants.Parsed()

	// Aliases must be granted using their canonical name.
	if alias := upstrGrts.Aliases[config.Privilege.ValueString()]; alias != "" {
		diags.AddAttributeError(
			path.Root("privilege_name"),
			"Cannot use alias",
			fmt.Sprintf("%q is an alias for %q. Please use %q instead", config.Privilege.ValueString(), alias, alias),
		)
		return
	}

	// Only the target attributes supported by the privilege's scope may be set.
	attrs, allAttrs, ok := grants.ScopeAttributesFor(config.Privilege.ValueString())
	if !ok {
		diags.AddAttributeError(
			path.Root("privilege_name"),
			"Unsupported Privilege",
			fmt.Sprintf("%q privilege_name is currently unsupported", config.Privilege.ValueString()),
		)
		return
	}

	checkAttr := func(attrName string, isSupported, isAllSupported, isSet bool) {
		if !isSet || isSupported {
			return
		}

		if isAllSupported {
			diags.AddAttributeWarning(
				path.Root(attrName),
				"Grant scope will be narrowed to supported grants",
				fmt.Sprintf("only %q descendants that support %q attribute will be granted", config.Privilege.ValueString(), attrName),
			)
			return
		}

		diags.AddAttributeError(
			path.Root(attrName),
			"Invalid Grant Privilege",
			fmt.Sprintf("%q must be null when 'privilege_name' is %q", attrName, config.Privilege.ValueString()),
		)
	}

	// CURRENT GRANTS targets the grantor's own privileges, so scope-based field requirements
	// (e.g. a GLOBAL privilege needing a null database_name) do not apply: the target can
	// legitimately be *.* for any privilege. Aliases and unsupported privileges stay blocked.
	if config.CurrentGrants.ValueBool() {
		return
	}

	checkAttr("database_name", attrs.Database, allAttrs.Database, !config.Database.IsNull())
	checkAttr("table_name", attrs.Table, allAttrs.Table, !config.Table.IsNull())
	checkAttr("column_name", attrs.Column, allAttrs.Column, !config.Column.IsNull())
}

func (r *Resource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config GrantPrivilege
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateScope(config, &resp.Diagnostics)
}

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// If the entire plan is null, the resource is planned for destruction.
		return
	}

	var config GrantPrivilege
	if !req.Config.Raw.IsNull() {
		resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
		validateScope(config, &resp.Diagnostics)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Only check replicated storage when cluster_name is set, to avoid
	// unnecessary connections (e.g. during terraform plan -refresh=false).
	if r.client != nil && !config.ClusterName.IsNull() {
		isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Could not check if service is using replicated storage",
				fmt.Sprintf("Skipping validation. If you are using replicated storage, please remove the 'cluster_name' attribute from your resource definition. Error: %+v", err),
			)
			return
		}

		// GrantPrivilege cannot specify 'cluster_name' or apply will fail.
		if isReplicatedStorage {
			resp.Diagnostics.AddWarning(
				"Invalid configuration",
				"Your ClickHouse cluster is using Replicated storage for grants, please remove the 'cluster_name' attribute from your GrantPrivilege resource definition if you encounter any errors.",
			)
		}
	}
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GrantPrivilege
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	grant := plan.toGrant()

	createdGrant, err := r.client.GrantPrivilege(ctx, grant, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Privilege Grant",
			"Could not create privilege grant, unexpected error: "+err.Error(),
		)
		return
	}

	if createdGrant == nil {
		existing, err := r.client.GetAllGrantsForGrantee(ctx, grant.GranteeUserName, grant.GranteeRoleName, plan.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error checking for existing overlapping privileges",
				"internal error while checking for existing overlapping privileges. Please try again",
			)
			return
		}

		overlappingExplanations := make([]string, 0)
		for _, e := range existing {
			if overlaps(plan, e) {
				// Prepare human-readable explanation of the overlap.
				overlappingExplanations = append(overlappingExplanations, explainOverlap(plan, e))
			}
		}

		if len(overlappingExplanations) > 0 {
			details := fmt.Sprintf(`While trying to apply this resource, we found some privileges already granted to the same grantee that are overlapping with this resource:
%s

This is a configuration error that prevents further actions. Please note that these privileges might have been granted outside terraform.`, strings.Join(overlappingExplanations, "\n"))

			resp.Diagnostics.AddError(
				"Overlapping Privilege",
				details,
			)
			return
		}

		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Privilege Grant",
			"The grant operation was successful but it didn't create the expected entry in system.grants table. This normally means there is an already granted privilege to the same grantee that already includes the one you tried to apply.",
		)
		return
	}

	state := toState(*createdGrant, plan.ClusterName)
	// current_grants is config-only: ClickHouse does not return it, so carry it forward.
	state.CurrentGrants = plan.CurrentGrants

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GrantPrivilege
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	grantPrivilege := state.toGrant()

	grant, err := r.client.GetGrantPrivilege(ctx, &grantPrivilege, state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Privilege Grant",
			"Could not read privilege grant, unexpected error: "+err.Error(),
		)
		return
	}

	if grant != nil {
		newState := toState(*grant, state.ClusterName)
		// current_grants is config-only: ClickHouse does not return it, so carry it forward.
		newState.CurrentGrants = state.CurrentGrants
		diags = resp.State.Set(ctx, &newState)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.State.RemoveResource(ctx)
	}
}

func (r *Resource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	panic("Update of grant privilege resource is not supported")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GrantPrivilege
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RevokeGrantPrivilege(ctx, state.toGrant(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Privilege Grant",
			"Could not delete privilege grant, unexpected error: "+err.Error(),
		)
		return
	}
}
