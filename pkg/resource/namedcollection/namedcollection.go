package namedcollection

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

// nonBlank rejects empty and whitespace-only key names, which ClickHouse cannot store.
var nonBlank = regexp.MustCompile(`\S`)

// secretKeyNamesKey names the private-state entry holding the names of the keys
// written from secret_keys_wo. Write-only values are absent from state, so
// without this Read would see them as added out of band and plan their removal.
const secretKeyNamesKey = "secret_key_names"

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
				Optional:    true,
				ElementType: types.StringType,
				Description: "Map of key/value pairs stored in the named collection, in the terraform state. Values from variables marked 'sensitive = true' are redacted from CLI output. For secrets you don't want in state at all, use 'secret_keys_wo'.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(nonBlank, "must not be blank")),
					mapvalidator.AtLeastOneOf(path.MatchRoot("secret_keys_wo")),
				},
			},
			"secret_keys_wo": schema.MapAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				ElementType: types.StringType,
				Description: "Map of key/value pairs stored in the named collection but never written to the terraform state. Requires Terraform/OpenTofu >= 1.11. Bump 'secret_keys_wo_version' to re-apply the values, ClickHouse never returns them so the provider cannot detect that they changed.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(nonBlank, "must not be blank")),
					mapvalidator.AlsoRequires(path.MatchRoot("secret_keys_wo_version")),
				},
			},
			"secret_keys_wo_version": schema.Int32Attribute{
				Optional:    true,
				Description: "Version of 'secret_keys_wo'. Bump it to re-apply every write-only value.",
				Validators: []validator.Int32{
					int32validator.AlsoRequires(path.MatchRoot("secret_keys_wo")),
				},
			},
			"overridable_keys": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Names of keys to mark as OVERRIDABLE. Keys listed in neither 'overridable_keys' nor 'not_overridable_keys' use the server default, which comes from the 'allow_named_collection_override_by_default' setting.",
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

	if config.Keys.IsUnknown() || config.SecretKeysWO.IsUnknown() || config.OverridableKeys.IsUnknown() || config.NotOverridableKeys.IsUnknown() {
		return
	}

	plainNames := mapAttributeKeyNames(config.Keys)
	secretNames := mapAttributeKeyNames(config.SecretKeysWO)

	// Both maps write into the same ClickHouse key namespace.
	for name := range secretNames {
		if _, ok := plainNames[name]; ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("secret_keys_wo"),
				"Invalid Named Collection",
				fmt.Sprintf("key %q can't be set in both 'keys' and 'secret_keys_wo'", name),
			)
		}
	}

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
			_, isPlain := plainNames[name]
			_, isSecret := secretNames[name]
			if !isPlain && !isSecret {
				resp.Diagnostics.AddAttributeError(
					path.Root(attrName),
					"Invalid Named Collection",
					fmt.Sprintf("key %q is not defined in 'keys' or 'secret_keys_wo'", name),
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
	var plan, config NamedCollection
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

	keys, secretNames, diags := resolveKeys(ctx, plan, config)
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

	resp.Diagnostics.Append(setSecretKeyNames(ctx, resp.Private, secretNames)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Overridable flags can't be read back from ClickHouse, the plan is authoritative.
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

	stateKeys, diags := mapAttributeToGoMap(ctx, state.Keys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	secretNames, diags := getSecretKeyNames(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	newKeys := make(map[string]string)

	for name, value := range stateKeys {
		key, ok := collection.Keys[name]
		if !ok {
			// Key was deleted outside of terraform, dropping it from state makes
			// the next plan add it back.
			continue
		}

		// ClickHouse only returns values to users granted SHOW NAMED COLLECTIONS
		// SECRETS. When it does, they are authoritative and drift is detected.
		if key.Value == dbops.HiddenNamedCollectionValue {
			newKeys[name] = value
		} else {
			newKeys[name] = key.Value
		}
	}

	// Keys added outside of terraform show up so the next plan removes them. Keys
	// written from secret_keys_wo are ours even though they are absent from state.
	for name, key := range collection.Keys {
		if _, ok := stateKeys[name]; ok {
			continue
		}
		if _, ok := secretNames[name]; ok {
			continue
		}
		newKeys[name] = key.Value
	}

	var missingSecret bool
	for name := range secretNames {
		if _, ok := collection.Keys[name]; !ok {
			missingSecret = true
		}
	}

	state.Keys, diags = goMapToMapAttribute(ctx, newKeys)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// A write-only key vanished from ClickHouse. Its value is not in state, so the
	// only way to plan a re-apply is to make the version look changed.
	if missingSecret {
		state.SecretKeysWOVersion = types.Int32Null()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state, config NamedCollection
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

	// Write-only attributes are only populated in the config, so retrieving the config as well.
	diags = req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plannedKeys, plannedSecretNames, diags := resolveKeys(ctx, plan, config)
	resp.Diagnostics.Append(diags...)

	statePlainValues, diags := mapAttributeToGoMap(ctx, state.Keys)
	resp.Diagnostics.Append(diags...)

	stateSecretNames, diags := getSecretKeyNames(ctx, req.Private)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateFlagFor := flagResolver(state)
	statePlainKeys := keysWithFlags(statePlainValues, stateFlagFor)

	plannedSecret := make(map[string]struct{}, len(plannedSecretNames))
	for _, name := range plannedSecretNames {
		plannedSecret[name] = struct{}{}
	}

	// Write-only values never reach the state, so a bumped version is the only
	// signal that they changed.
	versionChanged := !plan.SecretKeysWOVersion.Equal(state.SecretKeysWOVersion)

	set := make(map[string]dbops.NamedCollectionKey)
	deleteKeys := make([]string, 0)

	for name := range statePlainKeys {
		if _, ok := plannedKeys[name]; !ok {
			deleteKeys = append(deleteKeys, name)
		}
	}
	for name := range stateSecretNames {
		if _, ok := plannedKeys[name]; !ok {
			deleteKeys = append(deleteKeys, name)
		}
	}

	for name, plannedKey := range plannedKeys {
		var (
			exists   bool
			changed  bool
			prevFlag *bool
		)

		if _, isSecret := plannedSecret[name]; isSecret {
			_, exists = stateSecretNames[name]
			prevFlag = stateFlagFor(name)
			changed = !exists || versionChanged || !equalFlags(prevFlag, plannedKey.Overridable)
		} else {
			var stateKey dbops.NamedCollectionKey
			stateKey, exists = statePlainKeys[name]
			prevFlag = stateKey.Overridable
			changed = !exists || stateKey.Value != plannedKey.Value || !equalFlags(prevFlag, plannedKey.Overridable)
		}

		if !changed {
			continue
		}

		set[name] = plannedKey

		// Resetting a key's overridable flag to the server default requires
		// deleting the key and re-adding it.
		if exists && prevFlag != nil && plannedKey.Overridable == nil {
			deleteKeys = append(deleteKeys, name)
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

	resp.Diagnostics.Append(setSecretKeyNames(ctx, resp.Private, plannedSecretNames)...)
	if resp.Diagnostics.HasError() {
		return
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

// resolveKeys merges the plain 'keys' from the plan with the write-only
// 'secret_keys_wo' values from the config and attaches the overridable flags.
// Both maps share a single key namespace in ClickHouse. The returned names are
// the write-only ones, to be recorded in private state.
func resolveKeys(ctx context.Context, plan NamedCollection, config NamedCollection) (map[string]dbops.NamedCollectionKey, []string, diag.Diagnostics) {
	var diags diag.Diagnostics

	plainValues, d := mapAttributeToGoMap(ctx, plan.Keys)
	diags.Append(d...)

	secretValues, d := mapAttributeToGoMap(ctx, config.SecretKeysWO)
	diags.Append(d...)

	if diags.HasError() {
		return nil, nil, diags
	}

	flagFor := flagResolver(plan)

	keys := keysWithFlags(plainValues, flagFor)
	for name, key := range keysWithFlags(secretValues, flagFor) {
		keys[name] = key
	}

	secretNames := make([]string, 0, len(secretValues))
	for name := range secretValues {
		secretNames = append(secretNames, name)
	}
	sort.Strings(secretNames)

	return keys, secretNames, diags
}

func keysWithFlags(values map[string]string, flagFor func(string) *bool) map[string]dbops.NamedCollectionKey {
	ret := make(map[string]dbops.NamedCollectionKey, len(values))
	for name, value := range values {
		ret[name] = dbops.NamedCollectionKey{Value: value, Overridable: flagFor(name)}
	}
	return ret
}

// flagResolver reports the OVERRIDABLE flag a model configures for a key name,
// nil when the key is in neither set and the server default applies.
func flagResolver(model NamedCollection) func(string) *bool {
	overridable := setAttributeValues(model.OverridableKeys)
	notOverridable := setAttributeValues(model.NotOverridableKeys)

	return func(name string) *bool {
		if _, ok := overridable[name]; ok {
			return new(true)
		}
		if _, ok := notOverridable[name]; ok {
			return new(false)
		}
		return nil
	}
}

// privateState is the subset of the framework's private state used here. The
// concrete type lives in an internal package, so it can't be named directly.
type privateState interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

func setSecretKeyNames(ctx context.Context, private privateState, names []string) diag.Diagnostics {
	var diags diag.Diagnostics

	encoded, err := json.Marshal(names)
	if err != nil {
		diags.AddError("Error Storing Named Collection Private State", fmt.Sprintf("%+v\n", err))
		return diags
	}

	return private.SetKey(ctx, secretKeyNamesKey, encoded)
}

func getSecretKeyNames(ctx context.Context, private privateState) (map[string]struct{}, diag.Diagnostics) {
	ret := make(map[string]struct{})

	encoded, diags := private.GetKey(ctx, secretKeyNamesKey)
	if diags.HasError() || encoded == nil {
		return ret, diags
	}

	var names []string
	if err := json.Unmarshal(encoded, &names); err != nil {
		diags.AddError("Error Reading Named Collection Private State", fmt.Sprintf("%+v\n", err))
		return ret, diags
	}

	for _, name := range names {
		ret[name] = struct{}{}
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
