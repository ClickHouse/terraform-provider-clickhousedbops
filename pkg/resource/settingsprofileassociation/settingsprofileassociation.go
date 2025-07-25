package settingsprofileassociation

import (
	"context"
	_ "embed"
	"fmt"

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

//go:embed settingsprofileassociation.md
var settingsprofileassociationResourceDescription string

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
	resp.TypeName = req.ProviderTypeName + "_settingsprofileassociation"
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
			"settings_profile_name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the settings profile to associate",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the SettingsProfileAssociation to associate the Settings profile to",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("user_name")),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the User to associate the Settings profile to",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("role_name")),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		MarkdownDescription: settingsprofileassociationResourceDescription,
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
			var config SettingsProfileAssociation
			diags := req.Config.Get(ctx, &config)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// SettingsProfileAssociation cannot specify 'cluster_name' or apply will fail.
			if !config.ClusterName.IsNull() {
				resp.Diagnostics.AddWarning(
					"Invalid configuration",
					"Your ClickHouse cluster is using Replicated storage, please remove the 'cluster_name' attribute from your resource definition if you encounter any errors.",
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
	var plan SettingsProfileAssociation
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := SettingsProfileAssociation{
		ClusterName:         plan.ClusterName,
		SettingsProfileName: plan.SettingsProfileName,
	}

	if !plan.RoleName.IsUnknown() && !plan.RoleName.IsNull() {
		// Assign settings profile to role
		role, err := r.client.FindRoleByName(ctx, plan.RoleName.ValueString(), plan.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Role",
				fmt.Sprintf("%+v\n", err),
			)

			return
		}

		if role == nil {
			resp.Diagnostics.AddError(
				"Invalid configuration",
				fmt.Sprintf("No role with name %s was found", plan.RoleName.ValueString()),
			)
			return
		}

		if !role.HasSettingProfile(plan.SettingsProfileName.ValueString()) {
			err = r.client.AssociateSettingsProfile(ctx, plan.SettingsProfileName.ValueString(), &role.ID, nil, plan.ClusterName.ValueStringPointer())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Associating Settings Profile to Role",
					fmt.Sprintf("%+v\n", err),
				)

				return
			}
		}

		state.RoleName = types.StringValue(role.Name)
		state.UserName = types.StringNull()

	} else if !plan.UserName.IsUnknown() && !plan.UserName.IsNull() {
		// Assign settings profile to user
		user, err := r.client.FindUserByName(ctx, plan.UserName.ValueString(), plan.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting User",
				fmt.Sprintf("%+v\n", err),
			)

			return
		}

		if user == nil {
			resp.Diagnostics.AddError(
				"Invalid configuration",
				fmt.Sprintf("No user with name %s was found", plan.UserName.ValueString()),
			)
			return
		}

		if !user.HasSettingProfile(plan.SettingsProfileName.ValueString()) {
			err = r.client.AssociateSettingsProfile(ctx, plan.SettingsProfileName.ValueString(), nil, &user.ID, plan.ClusterName.ValueStringPointer())
			if err != nil {
				resp.Diagnostics.AddError(
					"Error Associating Settings Profile to Role",
					fmt.Sprintf("%+v\n", err),
				)

				return
			}
		}

		state.RoleName = types.StringNull()
		state.UserName = types.StringValue(user.Name)

	} else {
		resp.Diagnostics.AddError(
			"Invalid configuration",
			"No role or user was specified",
		)
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SettingsProfileAssociation
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.RoleName.IsUnknown() && !state.RoleName.IsNull() {
		role, err := r.client.FindRoleByName(ctx, state.RoleName.ValueString(), state.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Role",
				fmt.Sprintf("%+v\n", err),
			)

			return
		}

		if role == nil || !role.HasSettingProfile(state.SettingsProfileName.ValueString()) {
			state.RoleName = types.StringNull()
			resp.State.RemoveResource(ctx)
			return
		}
	} else if !state.UserName.IsUnknown() && !state.UserName.IsNull() {
		user, err := r.client.FindUserByName(ctx, state.UserName.ValueString(), state.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting User",
				fmt.Sprintf("%+v\n", err),
			)

			return
		}

		if user == nil || !user.HasSettingProfile(state.SettingsProfileName.ValueString()) {
			state.UserName = types.StringNull()
			resp.State.RemoveResource(ctx)
			return
		}
	} else {
		resp.State.RemoveResource(ctx)
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("Update operation is not supported for clickhousedbops_settingsprofileassociation resource")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SettingsProfileAssociation
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var userId, roleId *string
	if !state.RoleName.IsUnknown() && !state.RoleName.IsNull() {
		role, err := r.client.FindRoleByName(ctx, state.RoleName.ValueString(), state.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting Role",
				fmt.Sprintf("%+v\n", err),
			)

			return
		}

		if role != nil {
			roleId = &role.ID
		}
	} else if !state.UserName.IsUnknown() && !state.UserName.IsNull() {
		user, err := r.client.FindUserByName(ctx, state.UserName.ValueString(), state.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Getting User",
				fmt.Sprintf("%+v\n", err),
			)

			return
		}

		if user != nil {
			userId = &user.ID
		}
	}

	err := r.client.DisassociateSettingsProfile(ctx, state.SettingsProfileName.ValueString(), roleId, userId, state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse SettingsProfileAssociation",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}
