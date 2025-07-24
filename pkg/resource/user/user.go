package user

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed user.md
var userResourceDescription string

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
	resp.TypeName = req.ProviderTypeName + "_user"
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
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The system-assigned ID for the user",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the user",
			},
			"password_sha256_hash_wo": schema.StringAttribute{
				Required:    true,
				Description: "SHA256 hash of the password to be set for the user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-fA-F0-9]{64}$`), "password_sha256_hash must be a valid SHA256 hash"),
				},
				WriteOnly: true,
			},
			"password_sha256_hash_wo_version": schema.Int32Attribute{
				Required:    true,
				Description: "Version of the password_sha256_hash_wo field. Bump this value to require a force update of the password on the user.",
				PlanModifiers: []planmodifier.Int32{
					int32planmodifier.RequiresReplace(),
				},
			},
			"settings_profile": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the settings profile to assign to the user",
			},
		},
		MarkdownDescription: userResourceDescription,
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
			var config User
			diags := req.Config.Get(ctx, &config)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			// User cannot specify 'cluster_name' or apply will fail.
			if !config.ClusterName.IsNull() {
				resp.Diagnostics.AddWarning(
					"Invalid configuration",
					"Your ClickHouse cluster seems to be using Replicated storage for users, please remove the 'cluster_name' attribute from your User resource definition if you encounter any errors.",
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
	var plan User
	var config User
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write-only attributes are only populated in the config, so retrieving the config as well.
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user := dbops.User{
		Name:               plan.Name.ValueString(),
		PasswordSha256Hash: config.PasswordSha256Hash.ValueString(),
		SettingsProfile:    config.SettingsProfile.ValueStringPointer(),
	}

	createdUser, err := r.client.CreateUser(ctx, user, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse User",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	state := User{
		ClusterName:               plan.ClusterName,
		ID:                        types.StringValue(createdUser.ID),
		Name:                      types.StringValue(createdUser.Name),
		PasswordSha256HashVersion: plan.PasswordSha256HashVersion,
		SettingsProfile:           types.StringPointerValue(createdUser.SettingsProfile),
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state User
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.GetUser(ctx, state.ID.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse User",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	if user != nil {
		state.Name = types.StringValue(user.Name)
		state.SettingsProfile = types.StringPointerValue(user.SettingsProfile)

		diags = resp.State.Set(ctx, &state)
		resp.Diagnostics.Append(diags...)
	} else {
		resp.State.RemoveResource(ctx)
	}
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state User
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

	user, err := r.client.UpdateUser(ctx, dbops.User{
		ID:              state.ID.ValueString(),
		Name:            plan.Name.ValueString(),
		SettingsProfile: plan.SettingsProfile.ValueStringPointer(),
	}, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating ClickHouse User",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	state.Name = types.StringValue(user.Name)
	state.SettingsProfile = types.StringPointerValue(user.SettingsProfile)
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state User
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUser(ctx, state.ID.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse User",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// req.ID can either be in the form <cluster name>:<user ref> or just <user ref>
	// user ref can either be the name or the UUID of the user.

	// Check if cluster name is specified
	ref := req.ID
	var clusterName *string
	if strings.Contains(req.ID, ":") {
		clusterName = &strings.Split(req.ID, ":")[0]
		ref = strings.Split(req.ID, ":")[1]
	}

	// Check if ref is a UUID
	_, err := uuid.Parse(ref)
	if err != nil {
		// Failed parsing UUID, try importing using the database name
		user, err := r.client.FindUserByName(ctx, ref, clusterName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Cannot find user",
				fmt.Sprintf("%+v\n", err),
			)
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), user.ID)...)
	} else {
		// User passed a UUID
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), ref)...)
	}

	if clusterName != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	}
}
