package settingsprofile_test

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
	resourceType = "clickhousedbops_settingsprofile"
	resourceName = "foo"
)

func TestRole_acceptance(t *testing.T) {
	clusterName := "cluster1"

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		name := attrs["name"]
		if name == "" {
			return false, fmt.Errorf("name attribute was not set")
		}
		profile, err := dbopsClient.GetSettingsProfile(ctx, name, clusterName)
		return profile != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		name := attrs["name"]
		if name == nil {
			return fmt.Errorf("name was nil")
		}

		profile, err := dbopsClient.GetSettingsProfile(ctx, name.(string), clusterName)
		if err != nil {
			return err
		}

		if profile == nil {
			return fmt.Errorf("settings profile named %q was not found", name)
		}

		// Check state fields are aligned with the role we retrieved from CH.
		if !nilcompare.NilCompare(attrs["inherit_profile"], profile.InheritProfile) {
			return fmt.Errorf("wrong value for inherit_profile attribute")
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		if len(profile.Settings) != len(attrs["settings"].([]interface{})) {
			return fmt.Errorf("invalid number of settings")
		}

		for _, setting := range profile.Settings {
			// Look for the same setting in the attributes
			stateSettings := attrs["settings"].([]interface{})
			found := false

			for _, stateSetting := range stateSettings {
				state := stateSetting.(map[string]interface{})
				if setting.Name == state["name"].(string) {
					found = true
					if !nilcompare.NilCompare(setting.Value, state["value"]) {
						return fmt.Errorf("invalid Value for setting %q", setting.Name)
					}

					if !nilcompare.NilCompare(setting.Min, state["min"]) {
						return fmt.Errorf("invalid Min for setting %q", setting.Name)
					}

					if !nilcompare.NilCompare(setting.Max, state["max"]) {
						return fmt.Errorf("invalid Max for setting %q", setting.Name)
					}

					if !nilcompare.NilCompare(setting.Writability, state["writability"]) {
						return fmt.Errorf("invalid Writability for setting %q", setting.Name)
					}
					break
				}
			}

			if !found {
				return fmt.Errorf("setting %q was not found in the state", setting.Name)
			}
		}

		return nil
	}

	tests := []runner.TestCase{
		{
			Name:     "Create Settings Profile using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithListAttribute("settings", []cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":  cty.StringVal("max_threads"),
						"value": cty.StringVal("100"),
					}),
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create Settings Profile using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithListAttribute("settings", []cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("max_threads"),
						"min":  cty.StringVal("100"),
					}),
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create Settings Profile using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithListAttribute("settings", []cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name": cty.StringVal("max_threads"),
						"max":  cty.StringVal("100"),
					}),
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create Settings Profile using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithListAttribute("settings", []cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":  cty.StringVal("max_threads"),
						"value": cty.StringVal("500"),
						"min":   cty.StringVal("100"),
						"max":   cty.StringVal("1000"),
					}),
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create Settings Profile using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithStringAttribute("cluster_name", clusterName).
				WithListAttribute("settings", []cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":        cty.StringVal("max_threads"),
						"value":       cty.StringVal("500"),
						"min":         cty.StringVal("100"),
						"max":         cty.StringVal("1000"),
						"writability": cty.StringVal("CONST"),
					}),
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create Settings Profile using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithStringAttribute("cluster_name", clusterName).
				WithListAttribute("settings", []cty.Value{
					cty.ObjectVal(map[string]cty.Value{
						"name":        cty.StringVal("max_threads"),
						"value":       cty.StringVal("500"),
						"min":         cty.StringVal("100"),
						"max":         cty.StringVal("1000"),
						"writability": cty.StringVal("WRITABLE"),
					}),
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	runner.RunTests(t, tests)
}
