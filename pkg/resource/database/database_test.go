package database_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
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
		database, err := dbopsClient.GetDatabase(ctx, &dbops.Database{UUID: attrs["uuid"], Name: name}, clusterName)
		return database != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		name := attrs["name"]
		if name == nil {
			return fmt.Errorf("name was nil")
		}

		database, err := dbopsClient.GetDatabase(ctx, &dbops.Database{Name: name.(string)}, clusterName)
		if err != nil {
			return err
		}

		if database == nil {
			return fmt.Errorf("database with name %q was not found", name)
		}

		if id, _ := attrs["uuid"].(string); !database.SameAs(dbops.Database{UUID: id, Name: name.(string)}) {
			return fmt.Errorf("expected uuid to be %q, was %q", database.UUID, id)
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

		if attrs["engine_parameters_wo"] != nil {
			return fmt.Errorf("engine_parameters_wo was written to state")
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
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create Database with an engine sharing the zero UUID using Native protocol on a single replica",
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
		{
			Name:        "Import of a Database missing on some replicas fails using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "http",
			ClusterName: &clusterName,
			SetupFunc: func(ctx context.Context, dbopsClient dbops.Client, _ *string) error {
				_, err := dbopsClient.CreateDatabase(ctx, dbops.Database{Name: "partial"}, nil)
				return err
			},
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "partial").
				WithImport(clusterName + ":partial").
				Build(),
			ResourceName:    resourceName,
			ResourceAddress: fmt.Sprintf("%s.%s", resourceType, resourceName),
			ExpectError:     regexp.MustCompile(`(?s)database\s+with\s+UUID\s+"[^"]+"\s+exists\s+on\s+1\s+of\s+2\s+replicas`),
		},
		{
			Name:        "Import of a Database with different UUIDs on replicas fails using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "native",
			ClusterName: &clusterName,
			SetupFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string) error {
				db, err := dbopsClient.CreateDatabase(ctx, dbops.Database{Name: "drifted"}, clusterName)
				if err != nil {
					return err
				}
				if err = dbopsClient.DeleteDatabase(ctx, *db, nil); err != nil {
					return err
				}
				_, err = dbopsClient.CreateDatabase(ctx, dbops.Database{Name: "drifted"}, nil)
				return err
			},
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "drifted").
				WithImport(clusterName + ":drifted").
				Build(),
			ResourceName:    resourceName,
			ResourceAddress: fmt.Sprintf("%s.%s", resourceType, resourceName),
			ExpectError:     regexp.MustCompile(`(?s)database\s+"drifted"\s+has\s+inconsistent\s+metadata\s+across\s+cluster\s+replicas`),
		},
		{
			Name:        "Import Database with the same UUID on all replicas using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "native",
			ClusterName: &clusterName,
			SetupFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string) error {
				_, err := dbopsClient.CreateDatabase(ctx, dbops.Database{Name: "oncluster"}, clusterName)
				return err
			},
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "oncluster").
				WithImport(clusterName + ":oncluster").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	tests = append(tests,
		runner.TestCase{
			Name:        "Import of a Database with different engines on replicas fails using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "native",
			ClusterName: &clusterName,
			SetupFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string) error {
				db, err := dbopsClient.CreateDatabase(ctx, dbops.Database{Name: "mixed", Engine: "Memory"}, clusterName)
				if err != nil {
					return err
				}
				if err = dbopsClient.DeleteDatabase(ctx, *db, nil); err != nil {
					return err
				}
				_, err = dbopsClient.CreateDatabase(ctx, dbops.Database{Name: "mixed", Engine: "Dictionary"}, nil)
				return err
			},
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "mixed").
				WithImport(clusterName + ":mixed").
				Build(),
			ResourceName:    resourceName,
			ResourceAddress: fmt.Sprintf("%s.%s", resourceType, resourceName),
			ExpectError:     regexp.MustCompile(`(?s)database\s+"mixed"\s+has\s+inconsistent\s+metadata\s+across\s+cluster\s+replicas`),
		},
	)

	s3Resource := func(name string, version int64, secret string) string {
		return resourcebuilder.New(resourceType, resourceName).
			WithStringAttribute("name", name).
			WithStringAttribute("engine", "S3").
			WithListAttribute("engine_arguments", []cty.Value{
				cty.StringVal("'http://localhost:1/bucket/'"),
				cty.StringVal("{access_key_id:String}"),
				cty.StringVal("{secret_access_key:String}"),
			}).
			WithMapAttribute("engine_parameters_wo", map[string]cty.Value{
				"access_key_id":     cty.StringVal("key-id"),
				"secret_access_key": cty.StringVal(secret),
			}).
			WithIntAttribute("engine_parameters_wo_version", version).
			Build()
	}
	s3Name := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	tests = append(tests, runner.TestCase{
		Name:                "Rotate write-only engine parameters of an S3 Database recreates it using HTTP protocol on a single replica",
		ChEnv:               map[string]string{"CONFIGFILE": "config-single.xml"},
		Protocol:            "http",
		Resource:            s3Resource(s3Name, 1, "secret"),
		UpdateResource:      new(s3Resource(s3Name, 2, "rotated")),
		UpdateExpectReplace: true,
		ResourceName:        resourceName,
		ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
		CheckNotExistsFunc:  checkNotExistsFunc,
		CheckAttributesFunc: checkAttributesFunc,
	})

	replicatedArguments := []cty.Value{cty.StringVal("'/clickhouse/databases/{database}'"), cty.StringVal("'shard1'"), cty.StringVal("'replica1'")}
	replicatedSettings := map[string]cty.Value{"logs_to_keep": cty.StringVal("500")}

	importedName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	createdName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	tests = append(tests,
		runner.TestCase{
			Name:     "Import Replicated Database and adopt its engine configuration in place using Native protocol",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			SetupFunc: func(ctx context.Context, dbopsClient dbops.Client, clusterName *string) error {
				_, err := dbopsClient.CreateDatabase(ctx, dbops.Database{
					Name:            importedName,
					Engine:          "Replicated",
					EngineArguments: []string{"'/clickhouse/databases/{database}'", "'shard1'", "'replica1'"},
					EngineSettings:  map[string]string{"logs_to_keep": "500"},
				}, clusterName)
				return err
			},
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", importedName).
				WithStringAttribute("engine", "Replicated").
				WithImport(importedName).
				Build(),
			UpdateResource: new(resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", importedName).
				WithStringAttribute("engine", "Replicated").
				WithListAttribute("engine_arguments", replicatedArguments).
				WithMapAttribute("engine_settings", replicatedSettings).
				Build()),
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributesFunc,
		},
		runner.TestCase{
			Name:     "Change engine settings of a Replicated Database recreates it using HTTP protocol",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", createdName).
				WithStringAttribute("engine", "Replicated").
				WithListAttribute("engine_arguments", replicatedArguments).
				Build(),
			UpdateResource: new(resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", createdName).
				WithStringAttribute("engine", "Replicated").
				WithListAttribute("engine_arguments", replicatedArguments).
				WithMapAttribute("engine_settings", replicatedSettings).
				Build()),
			UpdateExpectReplace: true,
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	)

	runner.RunTests(t, tests)
}
