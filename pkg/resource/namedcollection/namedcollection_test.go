package namedcollection_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/zclconf/go-cty/cty"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/nilcompare"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/resourcebuilder"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/runner"
)

const (
	resourceType = "clickhousedbops_named_collection"
	resourceName = "foo"
)

func TestNamedcollection_acceptance(t *testing.T) {
	clusterName := "cluster1"

	keys := map[string]cty.Value{
		"url":               cty.StringVal("https://example.com/"),
		"format":            cty.StringVal("CSV"),
		"secret_access_key": cty.StringVal("topsecret"),
	}

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		name := attrs["name"]
		if name == "" {
			return false, fmt.Errorf("name attribute was not set")
		}
		collection, err := dbopsClient.GetNamedCollection(ctx, name, clusterName)
		return collection != nil, err
	}

	// checkAttributes compares the state with the actual collection. secretKeyNames
	// are the keys written from secret_keys_wo: they must exist in ClickHouse and
	// must NOT be in the terraform state. Their values are not compared, they are
	// never in state and ClickHouse hides them from users without
	// SHOW NAMED COLLECTIONS SECRETS.
	checkAttributes := func(secretKeyNames ...string) func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		return func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
			name := attrs["name"]
			if name == nil {
				return fmt.Errorf("name was nil")
			}

			collection, err := dbopsClient.GetNamedCollection(ctx, name.(string), clusterName)
			if err != nil {
				return err
			}

			if collection == nil {
				return fmt.Errorf("named collection named %q was not found", name)
			}

			if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
				return fmt.Errorf("wrong value for cluster_name attribute")
			}

			if attrs["secret_keys_wo"] != nil {
				return fmt.Errorf("secret_keys_wo was written to state")
			}

			stateKeys := make(map[string]string)
			if attrs["keys"] != nil {
				for k, v := range attrs["keys"].(map[string]interface{}) {
					stateKeys[k] = v.(string)
				}
			}

			if len(stateKeys)+len(secretKeyNames) != len(collection.Keys) {
				return fmt.Errorf("expected %d keys, clickhouse has %d", len(stateKeys)+len(secretKeyNames), len(collection.Keys))
			}

			for k, v := range stateKeys {
				actual, ok := collection.Keys[k]
				if !ok {
					return fmt.Errorf("key %q not found in clickhouse named collection", k)
				}

				if actual.Value != dbops.HiddenNamedCollectionValue && actual.Value != v {
					return fmt.Errorf("wrong value for key %q", k)
				}
			}

			for _, k := range secretKeyNames {
				if _, ok := collection.Keys[k]; !ok {
					return fmt.Errorf("write-only key %q not found in clickhouse named collection", k)
				}
				if _, ok := stateKeys[k]; ok {
					return fmt.Errorf("write-only key %q was written to state", k)
				}
			}

			return nil
		}
	}

	// Changes a value, adds a key, removes a key, resets the OVERRIDABLE flag on
	// 'url' back to the server default and marks the new key NOT OVERRIDABLE.
	updateName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	updateResource := resourcebuilder.New(resourceType, resourceName).
		WithStringAttribute("name", updateName).
		WithMapAttribute("keys", map[string]cty.Value{
			"url":    cty.StringVal("https://updated.example.com/"),
			"format": cty.StringVal("CSV"),
			"region": cty.StringVal("us-east-1"),
		}).
		WithListAttribute("not_overridable_keys", []cty.Value{cty.StringVal("region")}).
		Build()

	// Rotates the write-only value by bumping the version.
	rotateName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	rotateResource := resourcebuilder.New(resourceType, resourceName).
		WithStringAttribute("name", rotateName).
		WithMapAttribute("keys", map[string]cty.Value{"host": cty.StringVal("127.0.0.1")}).
		WithMapAttribute("secret_keys_wo", map[string]cty.Value{"password": cty.StringVal("rotated")}).
		WithIntAttribute("secret_keys_wo_version", 2).
		Build()

	tests := []runner.TestCase{
		{
			Name:     "Create Named Collection using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithMapAttribute("keys", keys).
				WithListAttribute("overridable_keys", []cty.Value{cty.StringVal("url")}).
				WithListAttribute("not_overridable_keys", []cty.Value{cty.StringVal("secret_access_key")}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributes(),
		},
		{
			Name:     "Create Named Collection using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithMapAttribute("keys", keys).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributes(),
		},
		{
			Name:     "Create Named Collection using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithMapAttribute("keys", keys).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributes(),
		},
		{
			Name:     "Create Named Collection using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithMapAttribute("keys", keys).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributes(),
		},
		{
			Name:        "Create Named Collection using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithStringAttribute("cluster_name", clusterName).
				WithMapAttribute("keys", keys).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributes(),
		},
		{
			Name:        "Create Named Collection using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithStringAttribute("cluster_name", clusterName).
				WithMapAttribute("keys", keys).
				WithListAttribute("overridable_keys", []cty.Value{cty.StringVal("url")}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributes(),
		},
		{
			Name:     "Change keys and overridable flags in place",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", updateName).
				WithMapAttribute("keys", keys).
				WithListAttribute("overridable_keys", []cty.Value{cty.StringVal("url")}).
				WithListAttribute("not_overridable_keys", []cty.Value{cty.StringVal("secret_access_key")}).
				Build(),
			UpdateResource:        &updateResource,
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributes(),
		},
		{
			Name:     "Rotate write-only secret keys via version bump in place",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", rotateName).
				WithMapAttribute("keys", map[string]cty.Value{"host": cty.StringVal("127.0.0.1")}).
				WithMapAttribute("secret_keys_wo", map[string]cty.Value{"password": cty.StringVal("topsecret")}).
				WithIntAttribute("secret_keys_wo_version", 1).
				WithListAttribute("not_overridable_keys", []cty.Value{cty.StringVal("password")}).
				Build(),
			UpdateResource:        &rotateResource,
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributes("password"),
		},
		{
			Name:        "Create Named Collection with write-only secret keys on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithStringAttribute("cluster_name", clusterName).
				WithMapAttribute("keys", map[string]cty.Value{"host": cty.StringVal("127.0.0.1")}).
				WithMapAttribute("secret_keys_wo", map[string]cty.Value{"password": cty.StringVal("topsecret")}).
				WithIntAttribute("secret_keys_wo_version", 1).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributes("password"),
		},
	}

	runner.RunTests(t, tests)
}
