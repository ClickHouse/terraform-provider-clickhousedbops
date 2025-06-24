package grantrole

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed grantrole.md
var grantResourceDescription string

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
	resp.TypeName = req.ProviderTypeName + "_grant_role"
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
			"role_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the role to be granted",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"grantee_user_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the `user` to grant `role_name` to.",
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
				Description: "Name of the `role` to grant `role_name` to.",
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
			"admin_option": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "If true, the grantee will be able to grant `role_name` to other `users` or `roles`.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
		MarkdownDescription: grantResourceDescription,
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
			var config GrantRole
			diags := req.Config.Get(ctx, &config)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// GrantRole cannot specify 'cluster_name' or apply will fail.
			if !config.ClusterName.IsNull() {
				resp.Diagnostics.AddWarning(
					"Invalid configuration",
					"Your ClickHouse cluster is using Replicated storage for role grants, please remove the 'cluster_name' attribute from your GrantRole resource definition if you encounter any errors.",
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
	var plan GrantRole
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	grant := dbops.GrantRole{
		RoleName:        plan.RoleName.ValueString(),
		GranteeUserName: plan.GranteeUserName.ValueStringPointer(),
		GranteeRoleName: plan.GranteeRoleName.ValueStringPointer(),
		AdminOption:     plan.AdminOption.ValueBool(),
	}

	createdGrant, err := r.client.GrantRole(ctx, grant, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse Role Grant",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	state := GrantRole{
		ClusterName:     plan.ClusterName,
		RoleName:        types.StringValue(createdGrant.RoleName),
		GranteeUserName: types.StringPointerValue(createdGrant.GranteeUserName),
		GranteeRoleName: types.StringPointerValue(createdGrant.GranteeRoleName),
		AdminOption:     types.BoolValue(createdGrant.AdminOption),
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GrantRole
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	grant, err := r.client.GetGrantRole(ctx, state.RoleName.ValueString(), state.GranteeUserName.ValueStringPointer(), state.GranteeRoleName.ValueStringPointer(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse Role Grant",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	if grant != nil {
		state.RoleName = types.StringValue(grant.RoleName)
		state.GranteeUserName = types.StringPointerValue(grant.GranteeUserName)
		state.GranteeRoleName = types.StringPointerValue(grant.GranteeRoleName)
		state.AdminOption = types.BoolValue(grant.AdminOption)

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.State.RemoveResource(ctx)
	}
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("Update of grant resource is not supported")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GrantRole
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.RevokeGrantRole(ctx, state.RoleName.ValueString(), state.GranteeUserName.ValueStringPointer(), state.GranteeRoleName.ValueStringPointer(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse Role Grant",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}
