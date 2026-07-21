package settingsprofile_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/zclconf/go-cty/cty"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/nilcompare"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/resourcebuilder"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/runner"
)

const (
	resourceType = "clickhousedbops_settings_profile"
	resourceName = "foo"
)

func TestSettingsprofile_acceptance(t *testing.T) {
	clusterName := "cluster1"

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		id := attrs["id"]
		if id == "" {
			return false, fmt.Errorf("id attribute was not set")
		}
		profile, err := dbopsClient.GetSettingsProfile(ctx, id, clusterName)
		return profile != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		id := attrs["id"]
		if id == nil {
			return fmt.Errorf("id was nil")
		}

		name := attrs["name"]
		if name == nil {
			return fmt.Errorf("name was nil")
		}

		profile, err := dbopsClient.GetSettingsProfile(ctx, id.(string), clusterName)
		if err != nil {
			return err
		}

		if profile == nil {
			return fmt.Errorf("settings profile named %q was not found", name)
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		// Check inherit_from

		if attrs["inherit_from"] == nil && len(profile.InheritFrom) > 0 ||
			attrs["inherit_from"] != nil && len(profile.InheritFrom) == 0 {
			return fmt.Errorf("wrong value for inherit_from attribute")
		}

		if attrs["inherit_from"] != nil {
			attrsInheritFrom := make([]string, 0)
			for _, i := range attrs["inherit_from"].([]interface{}) {
				attrsInheritFrom = append(attrsInheritFrom, i.(string))
			}

			if !reflect.DeepEqual(profile.InheritFrom, attrsInheritFrom) {
				return fmt.Errorf("wrong value for inherit_from attribute")
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
				WithListAttribute("inherit_from", []cty.Value{cty.StringVal("default")}).
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
				WithListAttribute("inherit_from", []cty.Value{cty.StringVal("default")}).
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
				WithListAttribute("inherit_from", []cty.Value{cty.StringVal("default")}).
				WithStringAttribute("cluster_name", clusterName).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	runner.RunTests(t, tests)
}

// TestSettingsprofile_adopt_acceptance is a regression test for the state
// reconciliation failure where a settings profile exists in ClickHouse but is
// missing from terraform state (a previous apply created it, the post-create
// lookup missed, and the ID was never saved). Re-applying used to fail
// permanently with ClickHouse error code 493 (ACCESS_ENTITY_ALREADY_EXISTS);
// the provider now adopts the existing profile by name and records its ID in
// state.
func TestSettingsprofile_adopt_acceptance(t *testing.T) {
	makeAdoptCase := func(name string, configFile string, protocol string) runner.TestCase {
		profileName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
		var preCreatedID string

		return runner.TestCase{
			Name:     name,
			ChEnv:    map[string]string{"CONFIGFILE": configFile},
			Protocol: protocol,
			// Seed the out-of-band profile before terraform runs, simulating
			// the "created but never recorded in state" divergence.
			SetupFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string) error {
				profile, err := dbopsClient.CreateSettingsProfile(ctx, dbops.SettingsProfile{Name: profileName}, clusterName)
				if err != nil {
					return fmt.Errorf("pre-creating settings profile %q: %w", profileName, err)
				}
				preCreatedID = profile.ID
				return nil
			},
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", profileName).
				Build(),
			ResourceName:    resourceName,
			ResourceAddress: fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
				id := attrs["id"]
				if id == "" {
					return false, fmt.Errorf("id attribute was not set")
				}
				profile, err := dbopsClient.GetSettingsProfile(ctx, id, clusterName)
				return profile != nil, err
			},
			CheckAttributesFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
				id, _ := attrs["id"].(string)
				if id == "" {
					return fmt.Errorf("id attribute was not set")
				}

				if id != preCreatedID {
					return fmt.Errorf("expected terraform to adopt pre-existing settings profile %q, but state has id %q", preCreatedID, id)
				}

				name, _ := attrs["name"].(string)
				if name != profileName {
					return fmt.Errorf("expected name %q in state, got %q", profileName, name)
				}

				profile, err := dbopsClient.GetSettingsProfile(ctx, id, clusterName)
				if err != nil {
					return err
				}
				if profile == nil {
					return fmt.Errorf("settings profile named %q was not found", profileName)
				}

				return nil
			},
		}
	}

	tests := []runner.TestCase{
		makeAdoptCase(
			"Adopt existing Settings Profile using HTTP protocol on a cluster using replicated storage",
			"config-replicated.xml",
			"http",
		),
		makeAdoptCase(
			"Adopt existing Settings Profile using Native protocol on a cluster using replicated storage",
			"config-replicated.xml",
			"native",
		),
		makeAdoptCase(
			"Adopt existing Settings Profile using HTTP protocol on a single replica",
			"config-single.xml",
			"http",
		),
	}

	runner.RunTests(t, tests)
}
