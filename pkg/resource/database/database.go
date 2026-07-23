package database

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
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
			"engine": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Database engine name. When omitted, ClickHouse chooses its default engine. Changing the engine recreates the database.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "must be a valid ClickHouse engine name"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"engine_arguments": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Ordered SQL expressions passed to the database engine. Include ClickHouse quoting where required. Changing arguments recreates the database.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must not be blank")),
					listvalidator.AlsoRequires(path.MatchRoot("engine")),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"engine_settings": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Database engine settings as setting-name to SQL-expression pairs. Values must include ClickHouse quoting where required. Changing settings recreates the database.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "must be a valid ClickHouse setting name")),
					mapvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must not be blank")),
					mapvalidator.AlsoRequires(path.MatchRoot("engine")),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"engine_parameters_wo": schema.MapAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				ElementType: types.StringType,
				Description: "Write-only string parameters referenced by engine arguments or settings, such as {catalog_credential:String}. The provider safely quotes substituted values and redacts them from logs and errors; they are not stored in state. Requires Terraform/OpenTofu >= 1.11.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "must be a valid ClickHouse query parameter name")),
					mapvalidator.AlsoRequires(
						path.MatchRoot("engine"),
						path.MatchRoot("engine_parameters_wo_version"),
					),
				},
			},
			"engine_parameters_wo_version": schema.Int64Attribute{
				Optional:    true,
				Description: "Version of engine_parameters_wo. Bump this value to recreate the database with updated write-only parameters.",
				Validators: []validator.Int64{
					int64validator.AlsoRequires(path.MatchRoot("engine_parameters_wo")),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
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

	var config Database
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	database, diags := databaseFromPlan(ctx, plan, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.CreateDatabase(ctx, database, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	plan.UUID = types.StringValue(db.UUID)
	state, err := r.syncDatabaseState(ctx, plan)
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

	state, err := r.syncDatabaseState(ctx, plan)
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

	err := r.client.DeleteDatabase(ctx, plan.Name.ValueString(), plan.ClusterName.ValueStringPointer())
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
	if cluster, databaseRef, found := strings.Cut(req.ID, ":"); found {
		if cluster == "" || databaseRef == "" || strings.Contains(databaseRef, ":") {
			resp.Diagnostics.AddError("Invalid database import identifier", "Expected <database ref> or <cluster name>:<database ref>.")
			return
		}
		clusterName = &cluster
		ref = databaseRef
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
		if db == nil {
			resp.Diagnostics.AddError("Cannot find database", fmt.Sprintf("Database %q was not found.", ref))
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), db.UUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), db.Name)...)
	} else {
		// Resolve the UUID once and persist the logical name. Subsequent reads use the
		// name because local UUIDs (notably Atomic databases) may differ by replica.
		db, err := r.client.GetDatabase(ctx, ref, clusterName)
		if err != nil {
			resp.Diagnostics.AddError("Cannot find database", fmt.Sprintf("%+v\n", err))
			return
		}
		if db == nil {
			resp.Diagnostics.AddError(
				"Cannot find database",
				fmt.Sprintf("Database UUID %q was not found. For clustered Atomic databases, import by name because UUIDs can differ between replicas.", ref),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), db.UUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), db.Name)...)
	}

	if clusterName != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	}
}

// syncDatabaseState reads database settings from clickhouse and returns a DatabaseResourceModel
func (r *Resource) syncDatabaseState(ctx context.Context, current Database) (*Database, error) {
	clusterName := current.ClusterName.ValueStringPointer()
	// UUIDs are not a unique database identity in ClickHouse: local engines may
	// share the zero UUID, while clustered Atomic databases can have one UUID per
	// replica. Treat the optionally cluster-qualified name as the logical identity.
	db, err := r.client.FindDatabaseByName(ctx, current.Name.ValueString(), clusterName)
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

	current.ClusterName = types.StringPointerValue(clusterName)
	if current.UUID.IsNull() || current.UUID.IsUnknown() || current.UUID.ValueString() == "" {
		current.UUID = types.StringValue(db.UUID)
	}
	current.Name = types.StringValue(db.Name)
	current.Comment = comment
	current.Engine = types.StringValue(db.Engine)

	return &current, nil
}

func databaseFromPlan(ctx context.Context, plan, config Database) (dbops.Database, diag.Diagnostics) {
	var diags diag.Diagnostics
	database := dbops.Database{
		Name:    plan.Name.ValueString(),
		Comment: plan.Comment.ValueString(),
	}

	if !plan.Engine.IsNull() && !plan.Engine.IsUnknown() {
		database.Engine = plan.Engine.ValueString()
	}
	if !plan.EngineArguments.IsNull() && !plan.EngineArguments.IsUnknown() {
		diags.Append(plan.EngineArguments.ElementsAs(ctx, &database.EngineArguments, false)...)
	}
	if !plan.EngineSettings.IsNull() && !plan.EngineSettings.IsUnknown() {
		diags.Append(plan.EngineSettings.ElementsAs(ctx, &database.EngineSettings, false)...)
	}
	if !config.EngineParametersWO.IsNull() && !config.EngineParametersWO.IsUnknown() {
		diags.Append(config.EngineParametersWO.ElementsAs(ctx, &database.EngineParameters, false)...)
	}

	return database, diags
}
