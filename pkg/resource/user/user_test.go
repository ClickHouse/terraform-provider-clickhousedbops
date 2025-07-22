package user_test

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
	resourceType = "clickhousedbops_user"
	resourceName = "foo"
)

func TestUser_acceptance(t *testing.T) {
	clusterName := "cluster1"

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		id := attrs["id"]
		if id == "" {
			return false, fmt.Errorf("id attribute was not set")
		}
		user, err := dbopsClient.GetUser(ctx, id, clusterName)
		return user != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		id := attrs["id"]
		if id == nil {
			return fmt.Errorf("id was nil")
		}

		user, err := dbopsClient.GetUser(ctx, id.(string), clusterName)
		if err != nil {
			return err
		}

		if user == nil {
			return fmt.Errorf("user with id %q was not found", id)
		}

		// Check state fields are aligned with the user we retrieved from CH.
		if attrs["name"].(string) != user.Name {
			return fmt.Errorf("expected name to be %q, was %q", user.Name, attrs["name"].(string))
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		return nil
	}

	tests := []runner.TestCase{
		{
			Name:        "Create User using Native protocol on a single replica",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash_wo", "sha256", "changeme").
				WithIntAttribute("password_sha256_hash_wo_version", 1).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create User using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash_wo", "sha256", "changeme").
				WithIntAttribute("password_sha256_hash_wo_version", 1).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create User using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash_wo", "sha256", "changeme").
				WithIntAttribute("password_sha256_hash_wo_version", 1).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create User using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash_wo", "sha256", "changeme").
				WithIntAttribute("password_sha256_hash_wo_version", 1).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create User using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "native",
			ClusterName: &clusterName,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash_wo", "sha256", "changeme").
				WithIntAttribute("password_sha256_hash_wo_version", 1).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create User using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "http",
			ClusterName: &clusterName,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash_wo", "sha256", "changeme").
				WithIntAttribute("password_sha256_hash_wo_version", 1).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	runner.RunTests(t, tests)
}
