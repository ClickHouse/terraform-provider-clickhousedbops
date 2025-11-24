package user_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/factories"
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

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]any) error {
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
		{
			Name:        "Create User with new password_sha256_hash field (OpenTofu version < 1.11 compatibility)",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash", "sha256", "changeme").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create User with new password_sha256_hash field on cluster with localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			Protocol:    "native",
			ClusterName: &clusterName,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithFunction("password_sha256_hash", "sha256", "changeme").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	runner.RunTests(t, tests)
}

func TestUser_validation_acceptance(t *testing.T) {
	providers := factories.ProviderFactories()
	const sha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	tests := []resource.TestCase{
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				// test ensure you can't use both password_sha256_hash and password_sha256_hash_wo
				{
					Config: fmt.Sprintf(`
					resource "clickhousedbops_user" "test" {
						name                    = "testuser"
						password_sha256_hash    = "%s"
						password_sha256_hash_wo = "%s"
						password_sha256_hash_wo_version = 1
					}
				`, sha256, sha256),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination.*password_sha256_hash.*cannot be specified`),
				},
				// test ensure you can't use password_sha256_hash_wo without specifying password_sha256_hash_wo_version
				{
					Config: fmt.Sprintf(`
					resource "clickhousedbops_user" "test" {
						name                    = "testuser"
						password_sha256_hash_wo = "%s"
					}
				`, sha256),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination.*password_sha256_hash_wo_version.*must be specified when.*password_sha256_hash_wo.*is specified`),
				},
				// test ensure you can't use password_sha256_hash_wo_version without password_sha256_hash_wo
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name                            = "testuser"
						password_sha256_hash_wo_version = 1
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination.*password_sha256_hash_wo.*must be specified when.*password_sha256_hash_wo_version.*is specified`),
				},
			},
		},
	}

	for _, tt := range tests {
		resource.Test(t, tt)
	}
}
