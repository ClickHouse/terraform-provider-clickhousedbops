package namedcollection

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

//go:embed namedcollection.md
var namedCollectionResourceDescription string

var (
	_ resource.Resource                   = &Resource{}
	_ resource.ResourceWithConfigure      = &Resource{}
	_ resource.ResourceWithImportState    = &Resource{}
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
	resp.TypeName = req.ProviderTypeName + "_named_collection"
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the named collection. ClickHouse does not support renaming named collections, so changing this forces a replacement.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"keys": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Map of key/value pairs stored in the named collection. For credentials, use values from variables marked 'sensitive = true' so terraform redacts them from CLI output.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
			},
			"overridable_keys": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Names of keys to mark as OVERRIDABLE. Keys not listed in 'overridable_keys' or 'not_overridable_keys' use the server default.",
			},
			"not_overridable_keys": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Names of keys to mark as NOT OVERRIDABLE.",
			},
		},
		MarkdownDescription: namedCollectionResourceDescription,
	}
}

func (r *Resource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config NamedCollection
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Keys.IsUnknown() || config.OverridableKeys.IsUnknown() || config.NotOverridableKeys.IsUnknown() {
		return
	}

	keyNames := mapAttributeKeyNames(config.Keys)

	overridable := setAttributeValues(config.OverridableKeys)
	notOverridable := setAttributeValues(config.NotOverridableKeys)

	for name := range notOverridable {
		if _, ok := overridable[name]; ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("not_overridable_keys"),
				"Invalid Named Collection",
				fmt.Sprintf("key %q can't be set in both 'overridable_keys' and 'not_overridable_keys'", name),
			)
		}
	}

	checkKeyExists := func(attrName string, names map[string]struct{}) {
		for name := range names {
			if _, ok := keyNames[name]; !ok {
				resp.Diagnostics.AddAttributeError(
					path.Root(attrName),
					"Invalid Named Collection",
					fmt.Sprintf("key %q is not defined in 'keys'", name),
				)
			}
		}
	}
	checkKeyExists("overridable_keys", overridable)
	checkKeyExists("not_overridable_keys", notOverridable)
}

func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// If the entire plan is null, the resource is planned for destruction.
		return
	}

	if r.client != nil {
		var config NamedCollection
		diags := req.Config.Get(ctx, &config)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Only check replicated storage when cluster_name is set, to avoid
		// unnecessary connections (e.g. during terraform plan -refresh=false).
		if !config.ClusterName.IsNull() {
			isReplicatedStorage, err := r.client.IsReplicatedStorage(ctx)
			if err != nil {
				resp.Diagnostics.AddWarning(
					"Could not check if service is using replicated storage",
					fmt.Sprintf("Skipping validation. If you are using replicated storage, please remove the 'cluster_name' attribute from your resource definition. Error: %+v", err),
				)
				return
			}

			if isReplicatedStorage {
				resp.Diagnostics.AddWarning(
					"Invalid configuration",
					"Your ClickHouse cluster is using Replicated storage, please remove the 'cluster_name' attribute from your NamedCollection resource definition if you encounter any errors.",
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
	var plan NamedCollection
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	keys, diags := mergedKeys(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection := dbops.NamedCollection{
		Name: plan.Name.ValueString(),
		Keys: keys,
	}

	_, err := r.client.CreateNamedCollection(ctx, collection, plan.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating ClickHouse NamedCollection",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	// Values and overridable flags can't be read back from ClickHouse, the plan is authoritative.
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state NamedCollection
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	collection, err := r.client.GetNamedCollection(ctx, state.Name.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Reading ClickHouse NamedCollection",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}

	if collection == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Values can be hidden from system tables, so only the set of key names is
	// reconciled: values in state stay authoritative from the config.
	stateKeys, diags := mapAttributeToGoMap(ctx, state.Keys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newKeys := make(map[string]string)

	for name, value := range stateKeys {
		if _, ok := collection.Keys[name]; ok {
			newKeys[name] = value
		}
	}

	// Keys added outside of terraform show up so the next plan reports them.
	for name, key := range collection.Keys {
		if _, ok := stateKeys[name]; !ok {
			newKeys[name] = key.Value
		}
	}

	state.Keys, diags = goMapToMapAttribute(ctx, newKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state NamedCollection
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

	plannedKeys, diags := mergedKeys(ctx, plan)
	resp.Diagnostics.Append(diags...)
	stateKeys, diags := mergedKeys(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	set := make(map[string]dbops.NamedCollectionKey)
	deleteKeys := make([]string, 0)

	for name := range stateKeys {
		if _, ok := plannedKeys[name]; !ok {
			deleteKeys = append(deleteKeys, name)
		}
	}

	for name, plannedKey := range plannedKeys {
		stateKey, exists := stateKeys[name]
		if !exists || stateKey.Value != plannedKey.Value || !equalFlags(stateKey.Overridable, plannedKey.Overridable) {
			set[name] = plannedKey

			// Resetting a key's overridable flag to the server default requires
			// deleting the key and re-adding it.
			if exists && stateKey.Overridable != nil && plannedKey.Overridable == nil {
				deleteKeys = append(deleteKeys, name)
			}
		}
	}

	if len(set) > 0 || len(deleteKeys) > 0 {
		collection, err := r.client.UpdateNamedCollection(ctx, state.Name.ValueString(), set, deleteKeys, plan.ClusterName.ValueStringPointer())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error Updating ClickHouse NamedCollection",
				fmt.Sprintf("%+v\n", err),
			)
			return
		}
		if collection == nil {
			resp.State.RemoveResource(ctx)
			return
		}
	}

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state NamedCollection
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteNamedCollection(ctx, state.Name.ValueString(), state.ClusterName.ValueStringPointer())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Deleting ClickHouse NamedCollection",
			fmt.Sprintf("%+v\n", err),
		)
		return
	}
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// req.ID can either be in the form <cluster name>:<collection name> or just <collection name>
	name := req.ID
	var clusterName *string
	if strings.Contains(req.ID, ":") {
		clusterName = &strings.Split(req.ID, ":")[0]
		name = strings.Split(req.ID, ":")[1]
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), name)...)

	if clusterName != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_name"), clusterName)...)
	}
}

// mergedKeys combines 'keys' with the overridable flags into the dbops
// representation of the collection's keys.
func mergedKeys(ctx context.Context, model NamedCollection) (map[string]dbops.NamedCollectionKey, diag.Diagnostics) {
	keys, diags := mapAttributeToGoMap(ctx, model.Keys)
	if diags.HasError() {
		return nil, diags
	}

	overridable := setAttributeValues(model.OverridableKeys)
	notOverridable := setAttributeValues(model.NotOverridableKeys)

	flagFor := func(name string) *bool {
		if _, ok := overridable[name]; ok {
			value := true
			return &value
		}
		if _, ok := notOverridable[name]; ok {
			value := false
			return &value
		}
		return nil
	}

	ret := make(map[string]dbops.NamedCollectionKey)
	for name, value := range keys {
		ret[name] = dbops.NamedCollectionKey{Value: value, Overridable: flagFor(name)}
	}

	return ret, diags
}

func mapAttributeToGoMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	ret := make(map[string]string)
	if m.IsNull() || m.IsUnknown() {
		return ret, nil
	}
	diags := m.ElementsAs(ctx, &ret, false)
	return ret, diags
}

func goMapToMapAttribute(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if len(m) == 0 {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

// mapAttributeKeyNames returns the key names of a map attribute, ignoring
// element values so it's safe to call on maps with unknown values.
func mapAttributeKeyNames(m types.Map) map[string]struct{} {
	ret := make(map[string]struct{})
	if m.IsNull() || m.IsUnknown() {
		return ret
	}
	for name := range m.Elements() {
		ret[name] = struct{}{}
	}
	return ret
}

func setAttributeValues(s types.Set) map[string]struct{} {
	ret := make(map[string]struct{})
	if s.IsNull() || s.IsUnknown() {
		return ret
	}
	for _, elem := range s.Elements() {
		if str, ok := elem.(types.String); ok && !str.IsNull() && !str.IsUnknown() {
			ret[str.ValueString()] = struct{}{}
		}
	}
	return ret
}

func equalFlags(a *bool, b *bool) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
