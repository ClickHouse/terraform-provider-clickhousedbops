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

	rotateName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	rotateUpdate := resourcebuilder.New(resourceType, resourceName).
		WithStringAttribute("name", rotateName).
		WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
			auth.WithBlock("sha256_hash", func(m *resourcebuilder.BlockBuilder) {
				m.WithFunction("value_wo", "sha256", "changeme2").WithIntAttribute("value_wo_version", 2)
			})
		}).
		Build()

	sslName := acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
	sslUpdate := resourcebuilder.New(resourceType, resourceName).
		WithStringAttribute("name", sslName).
		WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
			auth.WithBlock("ssl_certificate", func(m *resourcebuilder.BlockBuilder) {
				m.WithStringAttribute("common_name", "cn_b")
			})
		}).
		Build()

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
		{
			Name:        "Create user with sha256_hash auth block (write-only)",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
					auth.WithBlock("sha256_hash", func(m *resourcebuilder.BlockBuilder) {
						m.WithFunction("value_wo", "sha256", "changeme").
							WithIntAttribute("value_wo_version", 1)
					})
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create user with ssl_certificate auth block",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
					auth.WithBlock("ssl_certificate", func(m *resourcebuilder.BlockBuilder) {
						m.WithStringAttribute("common_name", "my_cn")
					})
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create user with multiple auth methods",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
					auth.WithBlock("ssl_certificate", func(m *resourcebuilder.BlockBuilder) {
						m.WithStringAttribute("common_name", "my_cn")
					})
					auth.WithBlock("bcrypt_password", func(m *resourcebuilder.BlockBuilder) {
						m.WithStringAttribute("value", "secret")
					})
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create user with no_password auth block",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)).
				WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
					auth.WithBlock("no_password", func(_ *resourcebuilder.BlockBuilder) {})
				}).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Rotate sha256_hash write-only value via version bump in place",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", rotateName).
				WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
					auth.WithBlock("sha256_hash", func(m *resourcebuilder.BlockBuilder) {
						m.WithFunction("value_wo", "sha256", "changeme").
							WithIntAttribute("value_wo_version", 1)
					})
				}).
				Build(),
			UpdateResource:        &rotateUpdate,
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributesFunc,
		},
		{
			Name:        "Change ssl_certificate common_name in place",
			ChEnv:       map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol:    "native",
			ClusterName: nil,
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", sslName).
				WithBlock("auth", func(auth *resourcebuilder.BlockBuilder) {
					auth.WithBlock("ssl_certificate", func(m *resourcebuilder.BlockBuilder) {
						m.WithStringAttribute("common_name", "cn_a")
					})
				}).
				Build(),
			UpdateResource:        &sslUpdate,
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributesFunc,
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
		// auth: no_password cannot be combined with another method
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
						auth {
							no_password {}
							plaintext_password { value = "p" }
						}
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination`),
				},
			},
		},
		// auth: no_password cannot be combined with a legacy password field
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: fmt.Sprintf(`
					resource "clickhousedbops_user" "test" {
						name                 = "testuser"
						password_sha256_hash = "%s"
						auth {
							no_password {}
						}
					}
				`, sha256),
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination`),
				},
			},
		},
		// auth: ssl_certificate needs exactly one of common_name / subject_alt_name (both set)
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
						auth {
							ssl_certificate {
								common_name      = "a"
								subject_alt_name = "b"
							}
						}
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination`),
				},
			},
		},
		// auth: ssl_certificate needs exactly one of common_name / subject_alt_name (neither set)
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
						auth {
							ssl_certificate {}
						}
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination`),
				},
			},
		},
		// auth: a secret method needs exactly one of value / value_wo (both set)
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
						auth {
							sha256_hash {
								value            = "x"
								value_wo         = "y"
								value_wo_version = 1
							}
						}
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Combination`),
				},
			},
		},
		// auth: sha256_hash value must be a valid SHA256 hash
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
						auth {
							sha256_hash { value = "nothex" }
						}
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value Match`),
				},
			},
		},
		// auth: double_sha1_hash value must be a valid SHA1 hash
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
						auth {
							double_sha1_hash { value = "nothex" }
						}
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value Match`),
				},
			},
		},
		// auth: bcrypt_hash value must be a valid bcrypt hash
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
						auth {
							bcrypt_hash { value = "notbcrypt" }
						}
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)Invalid Attribute Value Match`),
				},
			},
		},
		// no authentication configured at all
		{
			ProtoV6ProviderFactories: providers,
			Steps: []resource.TestStep{
				{
					Config: `
					resource "clickhousedbops_user" "test" {
						name = "testuser"
					}
				`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`(?s)(Invalid Attribute Combination|Missing Attribute Configuration|[Aa]t least one)`),
				},
			},
		},
	}

	for _, tt := range tests {
		resource.Test(t, tt)
	}
}
