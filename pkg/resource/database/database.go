package database

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed database.md
var databaseResourceDescription string

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithConfigure   = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
)

// NewResource is a helper function to simplify the provider implementation.
func NewResource() resource.Resource {
	return &Resource{}
}

// Resource is the resource implementation.
type Resource struct {
	client dbops.Client
}

// Metadata returns the resource type name.
func (r *Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the schema for the resource.
func (r *Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"cluster_name": schema.StringAttribute{
				Optional:    true,
				Description: "Name of the cluster to create the database into. If omitted, the database will be created on the replica hit by the query.\nThis field must be left null when using a ClickHouse Cloud cluster.\nShould be set when hitting a cluster with more than one replica.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"uuid": schema.StringAttribute{
				Computed:    true,
				Description: "The system-assigned UUID for the database",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the database",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"comment": schema.StringAttribute{
				Optional:    true,
				Description: "Comment associated with the database",
				Validators: []validator.String{
					// If user specifies the comment field, it can't be the empty string otherwise we get an error from terraform
					// due to the difference between null and empty string. User can always set this field to null or leave it out completely.
					stringvalidator.LengthAtLeast(1),
					stringvalidator.LengthAtMost(255),
				},
				PlanModifiers: []planmodifier.String{
					// Changing comment is not implemented: https://github.com/ClickHouse/ClickHouse/issues/73351
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		MarkdownDescription: databaseResourceDescription,
	}
}

func (r *Resource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(dbops.Client)
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan Database
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.CreateDatabase(ctx, dbops.Database{Name: plan.Name.ValueString(), Comment: plan.Comment.ValueString()}, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	state, err := r.syncDatabaseState(ctx, db.UUID, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error syncing database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	if state == nil {
		resp.Diagnostics.AddError(
			"Error syncing database",
			"failed retrieving database after creation",
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
	var plan Database
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state, err := r.syncDatabaseState(ctx, plan.UUID.ValueString(), plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error syncing database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	if state == nil {
		resp.State.RemoveResource(ctx)
	} else {
		diags = resp.State.Set(ctx, state)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	panic("unsupported")
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan Database
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDatabase(ctx, plan.UUID.ValueString(), plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// req.ID can either be in the form <cluster name>:<database ref> or just <database ref>
	// database ref can either be the name or the UUID of the database.

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
		db, err := r.client.FindDatabaseByName(ctx, ref, clusterName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Cannot find database",
				fmt.Sprintf("%+v\n", err),
			)
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), db.UUID)...)
	} else {
		// User passed a UUID
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), ref)...)
	}

	if clusterName != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	}
}

// syncDatabaseState reads database settings from clickhouse and returns a DatabaseResourceModel
func (r *Resource) syncDatabaseState(ctx context.Context, uuid string, clusterName *string) (*Database, error) {
	db, err := r.client.GetDatabase(ctx, uuid, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot get database")
	}

	if db == nil {
		// Database not found.
		return nil, nil
	}

	comment := types.StringNull()
	if db.Comment != "" {
		comment = types.StringValue(db.Comment)
	}

	state := &Database{
		ClusterName: types.StringPointerValue(clusterName),
		UUID:        types.StringValue(db.UUID),
		Name:        types.StringValue(db.Name),
		Comment:     comment,
	}

	return state, nil
}
