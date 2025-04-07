package database

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

	db, err := r.client.CreateDatabase(ctx, dbops.Database{Name: plan.Name.ValueString(), Comment: plan.Comment.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	state, err := r.syncDatabaseState(ctx, db.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error syncing database",
			fmt.Sprintf("%+v\n", err),
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

	state, err := r.syncDatabaseState(ctx, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error syncing database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
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

	err := r.client.DeleteDatabase(ctx, plan.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), req.ID)...)
}

// syncDatabaseState reads database settings from clickhouse and returns a DatabaseResourceModel
func (r *Resource) syncDatabaseState(ctx context.Context, uuid string) (*Database, error) {
	db, err := r.client.GetDatabase(ctx, uuid)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot get database")
	}

	comment := types.StringNull()
	if db.Comment != "" {
		comment = types.StringValue(db.Comment)
	}

	state := &Database{
		UUID:    types.StringValue(db.UUID),
		Name:    types.StringValue(db.Name),
		Comment: comment,
	}

	return state, nil
}
