package grantrole_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/nilcompare"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/resourcebuilder"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/runner"
)

const (
	resourceType = "clickhousedbops_grant_role"
	resourceName = "foo"

	roleName        = "role1"
	granteeRoleName = "grantee"
	granteeUserName = "user1"
)

func TestGrantRole_acceptance(t *testing.T) {
	clusterName := "cluster1"

	roleResource := resourcebuilder.New("clickhousedbops_role", roleName).WithStringAttribute("name", roleName)
	granteeRoleResource := resourcebuilder.
		New("clickhousedbops_role", granteeRoleName).
		WithStringAttribute("name", granteeRoleName)
	granteeUserResource := resourcebuilder.
		New("clickhousedbops_user", granteeUserName).
		WithStringAttribute("name", granteeUserName).
		WithFunction("password_sha256_hash_wo", "sha256", "test").
		WithIntAttribute("password_sha256_hash_wo_version", 1)

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		roleName := attrs["role_name"]
		if roleName == "" {
			return false, fmt.Errorf("role_name attribute was not set")
		}

		granteeUser := attrs["grantee_user_name"]
		granteeRole := attrs["grantee_role_name"]

		if granteeUser == "" && granteeRole == "" {
			return false, fmt.Errorf("both grantee_user_name and grantee_role_name attribute were not set")
		}

		var granteeUserName, granteeRoleName *string
		if granteeUser != "" {
			granteeUserName = &granteeUser
		}
		if granteeRole != "" {
			granteeRoleName = &granteeRole
		}

		grantrole, err := dbopsClient.GetGrantRole(ctx, roleName, granteeUserName, granteeRoleName, clusterName)
		return grantrole != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		roleName := attrs["role_name"]
		if roleName == nil {
			return fmt.Errorf("roleName was nil")
		}

		var granteeUserName, granteeRoleName *string
		if attrs["grantee_user_name"] != nil {
			s := attrs["grantee_user_name"].(string)
			granteeUserName = &s
		}

		if attrs["grantee_role_name"] != nil {
			s := attrs["grantee_role_name"].(string)
			granteeRoleName = &s
		}

		if granteeUserName == nil && granteeRoleName == nil {
			return fmt.Errorf("both grantee_user_name and grantee_role_name attribute were not set")
		}

		grantrole, err := dbopsClient.GetGrantRole(ctx, roleName.(string), granteeUserName, granteeRoleName, clusterName)
		if err != nil {
			return err
		}

		if grantrole == nil {
			return fmt.Errorf("grantrole was not found")
		}

		if attrs["role_name"].(string) != grantrole.RoleName {
			return fmt.Errorf("expected role_name to be %q, was %q", grantrole.RoleName, attrs["role_name"].(string))
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		if !nilcompare.NilCompare(grantrole.GranteeUserName, attrs["grantee_user_name"]) {
			return fmt.Errorf("wrong value for grantee_user_name attribute")
		}

		if !nilcompare.NilCompare(grantrole.GranteeRoleName, attrs["grantee_role_name"]) {
			return fmt.Errorf("wrong value for grantee_role_name attribute")
		}

		if grantrole.AdminOption != attrs["admin_option"].(bool) {
			return fmt.Errorf("wrong value for admin_option attribute")
		}

		return nil
	}

	tests := []runner.TestCase{
		// Single replica, Native
		{
			Name:     "Grant role to another role using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user with admin option using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("admin_option", true).
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Single replica, HTTP
		{
			Name:     "Grant role to another role using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user with admin option using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("admin_option", true).
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, native
		{
			Name:     "Grant role to another role using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user with admin option using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("admin_option", true).
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, http
		{
			Name:     "Grant role to another role using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant role to user with admin option using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("admin_option", true).
				AddDependency(roleResource.Build()).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, native
		{
			Name:        "Grant role to another role using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(roleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant role to user using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(roleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant role to user with admin option using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("admin_option", true).
				AddDependency(roleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, http
		{
			Name:        "Grant role to another role using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(roleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant role to user using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(roleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant role to user with admin option using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithResourceFieldReference("role_name", "clickhousedbops_role", roleName, "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("admin_option", true).
				AddDependency(roleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
	}

	runner.RunTests(t, tests)
}
