package setting_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/nilcompare"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/resourcebuilder"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/runner"
)

const (
	resourceType = "clickhousedbops_setting"
	resourceName = "foo"
)

func TestSettingsProfileSettings_acceptance(t *testing.T) {
	clusterName := "cluster1"

	settingProfileBuilder := resourcebuilder.New("clickhousedbops_settings_profile", "profile1").
		WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum))

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		profileID := attrs["settings_profile_id"]
		if profileID == "" {
			return false, fmt.Errorf("settings_profile_id attribute was not set")
		}
		name := attrs["name"]
		if name == "" {
			return false, fmt.Errorf("name attribute was not set")
		}
		profile, err := dbopsClient.GetSettingsProfile(ctx, profileID, clusterName)
		if err != nil {
			return false, err
		}

		if profile != nil {
			setting, err := dbopsClient.GetSetting(ctx, profile.ID, name, clusterName)
			if err != nil {
				return false, err
			}

			return setting != nil, err
		}

		return false, nil
	}
	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		profileID := attrs["settings_profile_id"]
		if profileID == "" {
			return fmt.Errorf("settings_profile_id attribute was not set")
		}
		name := attrs["name"]
		if name == nil {
			return fmt.Errorf("name was nil")
		}

		setting, err := dbopsClient.GetSetting(ctx, profileID.(string), name.(string), clusterName)
		if err != nil {
			return err
		}

		if setting == nil {
			return fmt.Errorf("setting named %q was not found", name)
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		if attrs["name"].(string) != setting.Name {
			return fmt.Errorf("expected name to be %q, was %q", setting.Name, attrs["name"].(string))
		}

		if !nilcompare.NilCompare(setting.Value, attrs["value"]) {
			return fmt.Errorf("wrong value for value attribute")
		}

		if !nilcompare.NilCompare(setting.Min, attrs["min"]) {
			return fmt.Errorf("wrong value for min attribute")
		}

		if !nilcompare.NilCompare(setting.Max, attrs["max"]) {
			return fmt.Errorf("wrong value for max attribute")
		}

		if !nilcompare.NilCompare(setting.Writability, attrs["writability"]) {
			return fmt.Errorf("wrong value for writability attribute")
		}

		return nil
	}

	tests := []runner.TestCase{
		{
			Name:     "Create Settings Profile Setting using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				AddDependency(settingProfileBuilder.Build()).
				WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
				WithStringAttribute("name", "max_threads").
				WithStringAttribute("value", "100").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create Settings Profile setting using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				AddDependency(settingProfileBuilder.Build()).
				WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
				WithStringAttribute("name", "max_threads").
				WithStringAttribute("min", "100").
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
				AddDependency(settingProfileBuilder.Build()).
				WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
				WithStringAttribute("name", "max_threads").
				WithStringAttribute("max", "100").
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
				AddDependency(settingProfileBuilder.Build()).
				WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
				WithStringAttribute("name", "max_threads").
				WithStringAttribute("value", "500").
				WithStringAttribute("min", "100").
				WithStringAttribute("max", "1000").
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
				AddDependency(settingProfileBuilder.WithStringAttribute("cluster_name", clusterName).Build()).
				WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
				WithStringAttribute("name", "max_threads").
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("value", "500").
				WithStringAttribute("min", "100").
				WithStringAttribute("max", "1000").
				WithStringAttribute("writability", "CONST").
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
				AddDependency(settingProfileBuilder.WithStringAttribute("cluster_name", clusterName).Build()).
				WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
				WithStringAttribute("name", "max_threads").
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("value", "500").
				WithStringAttribute("min", "100").
				WithStringAttribute("max", "1000").
				WithStringAttribute("writability", "WRITABLE").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	runner.RunTests(t, tests)
}
