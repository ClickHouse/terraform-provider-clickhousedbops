package database

import (
	"context"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// engineTrackedKey names the private-state entry marking engine configuration in state as applied by this provider.
const engineTrackedKey = "engine_tracked"

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &Resource{}
	_ resource.ResourceWithConfigure   = &Resource{}
	_ resource.ResourceWithImportState = &Resource{}
	_ resource.ResourceWithModifyPlan  = &Resource{}
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Description: "Database engine. When omitted, ClickHouse uses its default engine. Changing it forces a replacement.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "must be a valid ClickHouse engine name"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					// State written before engine support has no engine until refreshed.
					stringplanmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						resp.RequiresReplace = !req.StateValue.IsNull()
					}, "Changing the engine recreates the database.", "Changing the engine recreates the database."),
				},
			},
			"engine_arguments": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arguments of the database engine, as SQL expressions: quote string literals. Changing them forces a replacement.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must not be blank")),
					listvalidator.AlsoRequires(path.MatchRoot("engine")),
				},
				// ModifyPlan decides on replacement.
			},
			"engine_settings": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Settings of the database engine, as literals: quote string values. Changing them forces a replacement.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "must be a valid ClickHouse setting name")),
					mapvalidator.ValueStringsAre(stringvalidator.RegexMatches(regexp.MustCompile(`\S`), "must not be blank")),
					mapvalidator.AlsoRequires(path.MatchRoot("engine")),
				},
				// ModifyPlan decides on replacement.
			},
			"engine_parameters_wo": schema.MapAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				ElementType: types.StringType,
				Description: "Values of the query parameters, such as {secret_access_key:String}, used in 'engine_arguments' or 'engine_settings', never written to the terraform state. Requires Terraform/OpenTofu >= 1.11. Bump 'engine_parameters_wo_version' to re-apply the values, ClickHouse never returns them so the provider cannot detect that they changed.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`), "must be a valid ClickHouse query parameter name")),
					mapvalidator.AlsoRequires(
						path.MatchRoot("engine"),
						path.MatchRoot("engine_parameters_wo_version"),
					),
				},
			},
			"engine_parameters_wo_version": schema.Int32Attribute{
				Optional:    true,
				Description: "Version of 'engine_parameters_wo'. Bump it to re-apply every write-only value, which forces a replacement.",
				Validators: []validator.Int32{
					int32validator.AlsoRequires(path.MatchRoot("engine_parameters_wo")),
				},
				// ModifyPlan decides on replacement.
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

	resp.Diagnostics.Append(resp.Private.SetKey(ctx, engineTrackedKey, []byte("true"))...)
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

// ModifyPlan recreates the database on engine configuration changes, but adopts it in place when state predates it.
func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state Database
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tracked, diags := req.Private.GetKey(ctx, engineTrackedKey)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	attributes := []struct {
		path           path.Path
		planned, prior attr.Value
	}{
		{path.Root("engine_arguments"), plan.EngineArguments, state.EngineArguments},
		{path.Root("engine_settings"), plan.EngineSettings, state.EngineSettings},
		{path.Root("engine_parameters_wo_version"), plan.EngineParametersWOVersion, state.EngineParametersWOVersion},
	}
	var changed, adopted path.Paths
	for _, a := range attributes {
		switch {
		case a.planned.Equal(a.prior):
		case tracked == nil && a.prior.IsNull():
			adopted = append(adopted, a.path)
		default:
			changed = append(changed, a.path)
		}
	}

	if len(changed) > 0 {
		resp.RequiresReplace = append(resp.RequiresReplace, changed...)
		return
	}
	if !plan.ClusterName.Equal(state.ClusterName) || !plan.Name.Equal(state.Name) || !plan.Comment.Equal(state.Comment) || (!state.Engine.IsNull() && !plan.Engine.Equal(state.Engine)) {
		// Replaced anyway, so the configured engine is applied on create.
		return
	}
	for _, p := range adopted {
		resp.Diagnostics.AddAttributeWarning(
			p,
			"Engine configuration adopted without verification",
			"The database was imported or created by an older provider version, so its state has no engine configuration. "+
				"The configured value is recorded as-is: ClickHouse is not checked and the database is not changed. "+
				"Replace the resource to apply a different engine configuration.",
		)
	}
}

// Update only records engine configuration adopted by ModifyPlan, nothing changes in ClickHouse.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan Database
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var plan Database
	diags := req.State.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDatabase(ctx, plan.identity(), plan.ClusterName.ValueStringPointer())
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

	database := dbops.Database{Name: ref}
	if _, err := uuid.Parse(ref); err == nil {
		database = dbops.Database{UUID: ref}
	}

	db, err := r.client.GetDatabase(ctx, &database, clusterName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Cannot find database",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
	if db == nil {
		resp.Diagnostics.AddError("Cannot find database", fmt.Sprintf("database %q not found", ref))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), db.UUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), db.Name)...)

	if clusterName != nil {
		var dbName string
		resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("name"), &dbName)...)

		if err := r.client.CheckDatabaseReplicas(ctx, dbName, *clusterName); err != nil {
			resp.Diagnostics.AddError("Cannot import database", fmt.Sprintf("%+v\n", err))
			return
		}

		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	}
}

// syncDatabaseState reads database settings from clickhouse and returns a DatabaseResourceModel
func (r *Resource) syncDatabaseState(ctx context.Context, current Database) (*Database, error) {
	clusterName := current.ClusterName.ValueStringPointer()
	identity := current.identity()
	db, err := r.client.GetDatabase(ctx, &identity, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "cannot get database")
	}

	if db == nil || !db.SameAs(identity) {
		return nil, nil
	}

	comment := types.StringNull()
	if db.Comment != "" {
		comment = types.StringValue(db.Comment)
	}

	current.ClusterName = types.StringPointerValue(clusterName)
	if current.UUID.ValueString() == "" {
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
