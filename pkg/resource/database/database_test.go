package database_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/zclconf/go-cty/cty"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/nilcompare"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/resourcebuilder"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/runner"
)

const (
	resourceType = "clickhousedbops_database"
	resourceName = "foo"
)

func TestDatabase_acceptance(t *testing.T) {
	clusterName := "cluster1"

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		name := attrs["name"]
		if name == "" {
			return false, fmt.Errorf("name attribute was not set")
		}
		database, err := dbopsClient.FindDatabaseByName(ctx, name, clusterName)
		return database != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		name := attrs["name"]
		if name == nil {
			return fmt.Errorf("name was nil")
		}

		database, err := dbopsClient.FindDatabaseByName(ctx, name.(string), clusterName)
		if err != nil {
			return err
		}

		if database == nil {
			return fmt.Errorf("database with name %q was not found", name)
		}

		// Check state fields are aligned with the database we retrieved from CH.
		if attrs["name"].(string) != database.Name {
			return fmt.Errorf("expected name to be %q, was %q", database.Name, attrs["name"].(string))
		}

		if attrs["engine"].(string) != database.Engine {
			return fmt.Errorf("expected engine to be %q, was %q", database.Engine, attrs["engine"].(string))
		}

		var comment *string
		if database.Comment != "" {
			comment = &database.Comment
		}
		if !nilcompare.NilCompare(comment, attrs["comment"]) {
			return fmt.Errorf("wrong value for comment attribute")
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		return nil
	}

	tests := []runner.TestCase{
		{
			Name:     "Create Database using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithStringAttribute("engine", "Memory").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create Database using HTTP protocol on a single replica",
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
			Name:        "Create Database using Native protocol on a cluster using replicated storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-replicated.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithStringAttribute("comment", "test").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create Database using HTTP protocol on a cluster using replicated storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-replicated.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create Database using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "native",
			ClusterName: &clusterName,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create Database using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "http",
			ClusterName: &clusterName,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	// Import a Replicated database created outside of Terraform, then configure the engine
	// arguments and settings it was created with.
	importReplicatedCase := func(name string, configuredSettings map[string]string, expectAction plancheck.ResourceActionType) runner.TestCase {
		databaseName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
		arguments := []string{"'/clickhouse/databases/{database}'", "'shard1'", "'replica1'"}

		settings := make(map[string]cty.Value, len(configuredSettings))
		for k, v := range configuredSettings {
			settings[k] = cty.StringVal(v)
		}

		return runner.TestCase{
			Name:     name,
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			SetupFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string) error {
				_, err := dbopsClient.CreateDatabase(ctx, dbops.Database{
					Name:            databaseName,
					Engine:          "Replicated",
					EngineArguments: arguments,
					EngineSettings:  map[string]string{"logs_to_keep": "500"},
				}, clusterName)
				return err
			},
			ImportID:     databaseName,
			ExpectAction: expectAction,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", databaseName).
				WithStringAttribute("engine", "Replicated").
				WithListAttribute("engine_arguments", []cty.Value{
					cty.StringVal(arguments[0]), cty.StringVal(arguments[1]), cty.StringVal(arguments[2]),
				}).
				WithMapAttribute("engine_settings", settings).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		}
	}
	tests = append(tests,
		importReplicatedCase(
			"Import Replicated database and adopt matching engine configuration in place",
			map[string]string{"logs_to_keep": "500"},
			plancheck.ResourceActionUpdate,
		),
		importReplicatedCase(
			"Import Replicated database and replace it for different engine configuration",
			map[string]string{"logs_to_keep": "600"},
			plancheck.ResourceActionReplace,
		),
	)

	runner.RunTests(t, tests)
}
