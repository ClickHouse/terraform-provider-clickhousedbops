package rowpolicy_test

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
	resourceType = "clickhousedbops_row_policy"
	resourceName = "foo"

	granteeRoleName = "grantee"
	granteeUserName = "user1"
)

func TestRowpolicy_acceptance(t *testing.T) {
	clusterName := "cluster1"

	granteeRoleResource := resourcebuilder.
		New("clickhousedbops_role", granteeRoleName).
		WithStringAttribute("name", granteeRoleName)
	granteeUserResource := resourcebuilder.
		New("clickhousedbops_user", granteeUserName).
		WithStringAttribute("name", granteeUserName).
		WithFunction("password_sha256_hash_wo", "sha256", "test").
		WithIntAttribute("password_sha256_hash_wo_version", 1)

	checkNotExistsFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		name := attrs["name"]
		if name == "" {
			return false, fmt.Errorf("name attribute was not set")
		}

		database := attrs["database_name"]
		if database == "" {
			return false, fmt.Errorf("database_name attribute was not set")
		}

		table := attrs["table_name"]
		if table == "" {
			return false, fmt.Errorf("table_name attribute was not set")
		}

		// Handle both old single-name fields and new list fields for backward compatibility
		var userNames []string
		var roleNames []string
		var granteeAll bool
		var granteeAllExcept []string

		if granteeUser := attrs["grantee_user_names"]; granteeUser != "" {
			userNames = []string{granteeUser}
		}
		if granteeRole := attrs["grantee_role_names"]; granteeRole != "" {
			roleNames = []string{granteeRole}
		}
		// Also support old field names for existing tests
		if granteeUser := attrs["grantee_user_name"]; granteeUser != "" && len(userNames) == 0 {
			userNames = []string{granteeUser}
		}
		if granteeRole := attrs["grantee_role_name"]; granteeRole != "" && len(roleNames) == 0 {
			roleNames = []string{granteeRole}
		}

		if len(userNames) == 0 && len(roleNames) == 0 && !granteeAll && len(granteeAllExcept) == 0 {
			return false, fmt.Errorf("no grantee specification provided")
		}

		rowPolicy := dbops.RowPolicy{
			Name:             name,
			Database:         database,
			Table:            table,
			GranteeUserNames: userNames,
			GranteeRoleNames: roleNames,
			GranteeAll:       granteeAll,
			GranteeAllExcept: granteeAllExcept,
		}

		rp, err := dbopsClient.GetRowPolicy(ctx, &rowPolicy, clusterName)
		return rp != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		name := attrs["name"].(string)
		if name == "" {
			return fmt.Errorf("name attribute was not set")
		}

		database := attrs["database_name"].(string)
		if database == "" {
			return fmt.Errorf("database_name attribute was not set")
		}

		table := attrs["table_name"].(string)
		if table == "" {
			return fmt.Errorf("table_name attribute was not set")
		}

		// Handle both old single-name fields and new list fields for backward compatibility
		var userNames []string
		var roleNames []string
		var granteeAll bool
		var granteeAllExcept []string

		// Try new list fields first
		if userNamesAttr, ok := attrs["grantee_user_names"]; ok && userNamesAttr != nil {
			if userNamesList, ok := userNamesAttr.([]interface{}); ok {
				for _, u := range userNamesList {
					if uStr, ok := u.(string); ok {
						userNames = append(userNames, uStr)
					}
				}
			}
		}
		if roleNamesAttr, ok := attrs["grantee_role_names"]; ok && roleNamesAttr != nil {
			if roleNamesList, ok := roleNamesAttr.([]interface{}); ok {
				for _, r := range roleNamesList {
					if rStr, ok := r.(string); ok {
						roleNames = append(roleNames, rStr)
					}
				}
			}
		}

		// Support old field names for backward compatibility
		if granteeUserAttr, ok := attrs["grantee_user_name"]; ok && granteeUserAttr != nil && len(userNames) == 0 {
			if userStr, ok := granteeUserAttr.(string); ok && userStr != "" {
				userNames = []string{userStr}
			}
		}
		if granteeRoleAttr, ok := attrs["grantee_role_name"]; ok && granteeRoleAttr != nil && len(roleNames) == 0 {
			if roleStr, ok := granteeRoleAttr.(string); ok && roleStr != "" {
				roleNames = []string{roleStr}
			}
		}

		if granteeAllAttr, ok := attrs["grantee_all"]; ok && granteeAllAttr != nil {
			granteeAll = granteeAllAttr.(bool)
		}

		if granteeAllExceptAttr, ok := attrs["grantee_all_except"]; ok && granteeAllExceptAttr != nil {
			if granteeAllExceptList, ok := granteeAllExceptAttr.([]interface{}); ok {
				for _, e := range granteeAllExceptList {
					if eStr, ok := e.(string); ok {
						granteeAllExcept = append(granteeAllExcept, eStr)
					}
				}
			}
		}

		isRestrictive := false
		if attrs["is_restrictive"] != nil {
			isRestrictive = attrs["is_restrictive"].(bool)
		}

		rowPolicy := dbops.RowPolicy{
			Name:             name,
			Database:         database,
			Table:            table,
			GranteeUserNames: userNames,
			GranteeRoleNames: roleNames,
			GranteeAll:       granteeAll,
			GranteeAllExcept: granteeAllExcept,
		}

		rp, err := dbopsClient.GetRowPolicy(ctx, &rowPolicy, clusterName)
		if err != nil {
			return err
		}

		if rp == nil {
			return fmt.Errorf("row policy was not found")
		}

		if name != rp.Name {
			return fmt.Errorf("expected name to be %q, was %q", name, rp.Name)
		}

		if database != rp.Database {
			return fmt.Errorf("expected database_name to be %q, was %q", database, rp.Database)
		}

		if table != rp.Table {
			return fmt.Errorf("expected table_name to be %q, was %q", table, rp.Table)
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		// Validate grantee specification
		if len(rp.GranteeUserNames) != len(userNames) {
			return fmt.Errorf("expected %d user names, got %d", len(userNames), len(rp.GranteeUserNames))
		}
		for i, expected := range userNames {
			if i >= len(rp.GranteeUserNames) || rp.GranteeUserNames[i] != expected {
				return fmt.Errorf("expected user name %q at position %d, got %q", expected, i, rp.GranteeUserNames[i])
			}
		}

		if len(rp.GranteeRoleNames) != len(roleNames) {
			return fmt.Errorf("expected %d role names, got %d", len(roleNames), len(rp.GranteeRoleNames))
		}
		for i, expected := range roleNames {
			if i >= len(rp.GranteeRoleNames) || rp.GranteeRoleNames[i] != expected {
				return fmt.Errorf("expected role name %q at position %d, got %q", expected, i, rp.GranteeRoleNames[i])
			}
		}

		if rp.GranteeAll != granteeAll {
			return fmt.Errorf("expected grantee_all to be %v, was %v", granteeAll, rp.GranteeAll)
		}

		if len(rp.GranteeAllExcept) != len(granteeAllExcept) {
			return fmt.Errorf("expected %d all_except values, got %d", len(granteeAllExcept), len(rp.GranteeAllExcept))
		}

		if rp.IsRestrictive != isRestrictive {
			return fmt.Errorf("expected is_restrictive to be %v, was %v", isRestrictive, rp.IsRestrictive)
		}

		return nil
	}

	tests := []runner.TestCase{
		// Single replica, Native
		{
			Name:     "Create permissive row policy for role using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_role_names", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name != 'system'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Single replica, HTTP
		{
			Name:     "Create permissive row policy for role using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'default'").
				WithListResourceFieldReference("grantee_role_names", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, native
		{
			Name:     "Create permissive row policy for role using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_role_names", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'system'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name != 'system'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, http
		{
			Name:     "Create permissive row policy for role using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'default'").
				WithListResourceFieldReference("grantee_role_names", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create permissive row policy for user using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create restrictive row policy for user using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, native
		{
			Name:        "Create permissive row policy for role using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_role_names", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create permissive row policy for user using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'system'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create restrictive row policy for user using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name != 'system'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, http
		{
			Name:        "Create permissive row policy for role using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("select_filter", "database = 'default'").
				WithListResourceFieldReference("grantee_role_names", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create permissive row policy for user using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Create restrictive row policy for user with grant option using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_user_names", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
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

func stringPtr(s string) *string {
	return &s
}
