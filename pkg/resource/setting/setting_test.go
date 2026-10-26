package setting_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/zclconf/go-cty/cty"

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
	profileName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)

	settingProfileBuilder := resourcebuilder.New("clickhousedbops_settings_profile", "profile1").
		WithStringAttribute("name", profileName)

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

	withCreateTimeout := func(timeout string) string {
		return resourcebuilder.New(resourceType, resourceName).
			AddDependency(settingProfileBuilder.Build()).
			WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
			WithStringAttribute("name", "max_threads").
			WithStringAttribute("value", "100").
			WithMapAttribute("timeouts", map[string]cty.Value{"create": cty.StringVal(timeout)}).
			Build()
	}
	timeoutsCreate := withCreateTimeout("1m")
	timeoutsUpdate := withCreateTimeout("2m")

	// checkStoredAs checks ClickHouse stored value, min and max in its own format.
	checkStoredAs := func(stored map[string]interface{}) func(context.Context, dbops.Client, *string, map[string]interface{}) error {
		return func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
			setting, err := dbopsClient.GetSetting(ctx, attrs["settings_profile_id"].(string), attrs["name"].(string), clusterName)
			if err != nil {
				return err
			}
			if setting == nil {
				return fmt.Errorf("setting named %q was not found", attrs["name"])
			}

			for name, got := range map[string]*string{"value": setting.Value, "min": setting.Min, "max": setting.Max} {
				if !nilcompare.NilCompare(got, stored[name]) {
					return fmt.Errorf("expected %s stored as %v", name, stored[name])
				}
			}

			return nil
		}
	}
	reformattedSizes := resourcebuilder.New(resourceType, resourceName).
		AddDependency(settingProfileBuilder.Build()).
		WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
		WithStringAttribute("name", "max_memory_usage").
		WithStringAttribute("value", "10G").
		WithStringAttribute("min", "1k").
		WithStringAttribute("max", "25G").
		Build()
	reformattedCustom := resourcebuilder.New(resourceType, resourceName).
		AddDependency(settingProfileBuilder.Build()).
		WithResourceFieldReference("settings_profile_id", "clickhousedbops_settings_profile", "profile1", "id").
		WithStringAttribute("name", "SQL_tenant").
		WithStringAttribute("value", "").
		Build()

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
		{
			Name:                  "Change Settings Profile setting timeouts in place",
			ChEnv:                 map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:              "native",
			Resource:              timeoutsCreate,
			UpdateResource:        &timeoutsUpdate,
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributesFunc,
		},
		{
			Name:                "Keep configured size values ClickHouse expands using Native protocol on a single replica",
			ChEnv:               map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:            "native",
			Resource:            reformattedSizes,
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkStoredAs(map[string]interface{}{"value": "10000000000", "min": "1000", "max": "25000000000"}),
		},
		{
			Name:                "Keep configured custom setting value ClickHouse quotes using HTTP protocol on a single replica",
			ChEnv:               map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:            "http",
			Resource:            reformattedCustom,
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkStoredAs(map[string]interface{}{"value": "''"}),
		},
		{
			Name:                "Replace setting changed out of band using Native protocol on a single replica",
			ChEnv:               map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:            "native",
			Resource:            reformattedSizes,
			UpdateResource:      &reformattedSizes,
			UpdateExpectReplace: true,
			UpdateSetupFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string) error {
				profile, err := dbopsClient.FindSettingsProfileByName(ctx, profileName, clusterName)
				if err != nil {
					return err
				}
				if profile == nil {
					return fmt.Errorf("settings profile %q was not found", profileName)
				}
				if err := dbopsClient.DeleteSetting(ctx, profile.ID, "max_memory_usage", clusterName); err != nil {
					return err
				}
				_, err = dbopsClient.CreateSetting(ctx, profile.ID, dbops.Setting{Name: "max_memory_usage", Value: new("20G")}, clusterName, time.Minute)
				return err
			},
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkStoredAs(map[string]interface{}{"value": "10000000000", "min": "1000", "max": "25000000000"}),
		},
	}

	runner.RunTests(t, tests)
}
