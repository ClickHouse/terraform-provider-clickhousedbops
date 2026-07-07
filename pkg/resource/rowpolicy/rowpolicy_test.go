package rowpolicy_test

import (
	"context"
	"fmt"
	"regexp"
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

		// GetRowPolicy looks the policy up by name/database/table; grantees are not needed.
		rowPolicy := dbops.RowPolicy{
			Name:     name,
			Database: database,
			Table:    table,
		}

		rp, err := dbopsClient.GetRowPolicy(ctx, &rowPolicy, clusterName)
		return rp != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		id := attrs["id"].(string)
		if id == "" {
			return fmt.Errorf("id attribute was not set")
		}

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

		var granteeNames []string
		var granteeAllExcept []string

		if granteeNamesAttr, ok := attrs["grantee_names"]; ok && granteeNamesAttr != nil {
			if granteeNamesList, ok := granteeNamesAttr.([]interface{}); ok {
				for _, g := range granteeNamesList {
					if gStr, ok := g.(string); ok {
						granteeNames = append(granteeNames, gStr)
					}
				}
			}
		}

		// A present grantee_all_except (even empty) means "apply to all"; absent means named grantees.
		granteeAllExceptAttr, granteeAll := attrs["grantee_all_except"]
		granteeAll = granteeAll && granteeAllExceptAttr != nil
		if granteeAll {
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

		rp, err := dbopsClient.GetRowPolicyByID(ctx, id, clusterName)
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

		if len(rp.GranteeNames) != len(granteeNames) {
			return fmt.Errorf("expected %d grantees, got %d", len(granteeNames), len(rp.GranteeNames))
		}
		gotGrantees := make(map[string]bool, len(rp.GranteeNames))
		for _, g := range rp.GranteeNames {
			gotGrantees[g] = true
		}
		for _, expected := range granteeNames {
			if !gotGrantees[expected] {
				return fmt.Errorf("expected grantee %q to be present, got %v", expected, rp.GranteeNames)
			}
		}

		// A present grantee_all_except (empty or not) maps to apply_to_all=1 at the dbops layer.
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_role", granteeRoleName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("is_restrictive", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create row policy for all users except one using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithListResourceFieldReference("grantee_all_except", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Create duplicate row policy reports already exists with import hint using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: fmt.Sprintf("%s\n%s",
				resourcebuilder.New(resourceType, resourceName).
					WithStringAttribute("name", "dup_policy").
					WithStringAttribute("database_name", "system").
					WithStringAttribute("table_name", "databases").
					WithStringAttribute("select_filter", "1").
					WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
					AddDependency(granteeUserResource.Build()).
					Build(),
				resourcebuilder.New(resourceType, "bar").
					WithResourceFieldReference("name", resourceType, resourceName, "name").
					WithStringAttribute("database_name", "system").
					WithStringAttribute("table_name", "databases").
					WithStringAttribute("select_filter", "1").
					WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
					Build(),
			),
			ResourceName:    resourceName,
			ResourceAddress: fmt.Sprintf("%s.%s", resourceType, resourceName),
			ExpectError:     regexp.MustCompile("Import it with"),
		},
		{
			Name:     "Create row policy for all users with an empty except set using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "1").
				WithEmptyListAttribute("grantee_all_except").
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Update row policy filter and restrictiveness in place using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			UpdateResource: ptr(resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name != 'system'").
				WithBoolAttribute("is_restrictive", true).
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build()),
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributesFunc,
		},
		{
			Name:     "Rename row policy in place using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			UpdateResource: ptr(resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("name", "test_policy_renamed").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("select_filter", "name = 'default'").
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build()),
			UpdateExpectNoReplace: true,
			ResourceName:          resourceName,
			ResourceAddress:       fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:    checkNotExistsFunc,
			CheckAttributesFunc:   checkAttributesFunc,
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_role", granteeRoleName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_role", granteeRoleName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_role", granteeRoleName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_role", granteeRoleName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_role", granteeRoleName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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
				WithListResourceFieldReference("grantee_names", "clickhousedbops_user", granteeUserName, "name").
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

func ptr(s string) *string {
	return &s
}
