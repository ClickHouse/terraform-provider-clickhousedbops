package grantprivilege_test

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
	resourceType = "clickhousedbops_grant_privilege"
	resourceName = "foo"

	granteeRoleName = "grantee"
	granteeUserName = "user1"
)

func TestGrantprivilege_acceptance(t *testing.T) {
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
		accessType := attrs["privilege_name"]
		if accessType == "" {
			return false, fmt.Errorf("privilege_name attribute was not set")
		}

		granteeUser := attrs["grantee_user_name"]
		granteeRole := attrs["grantee_role_name"]

		if granteeUser == "" && granteeRole == "" {
			return false, fmt.Errorf("both grantee_user_name and grantee_role_name attribute were not set")
		}

		var database *string
		if attrs["database_name"] != "" {
			s := attrs["database_name"]
			database = &s
		}

		var table *string
		if attrs["table_name"] != "" {
			s := attrs["table_name"]
			table = &s
		}

		var column *string
		if attrs["column_name"] != "" {
			s := attrs["column_name"]
			column = &s
		}

		var granteeUserName, granteeRoleName *string
		if granteeUser != "" {
			granteeUserName = &granteeUser
		}
		if granteeRole != "" {
			granteeRoleName = &granteeRole
		}

		grantprivilege, err := dbopsClient.GetGrantPrivilege(ctx, accessType, database, table, column, granteeUserName, granteeRoleName, clusterName)
		return grantprivilege != nil, err
	}

	checkAttributesFunc := func(ctx context.Context, dbopsClient dbops.Client, clusterName *string, attrs map[string]interface{}) error {
		accessType := attrs["privilege_name"]
		if accessType == "" {
			return fmt.Errorf("privilege_name attribute was not set")
		}

		var database *string
		if attrs["database_name"] != nil {
			s := attrs["database_name"].(string)
			database = &s
		}

		var table *string
		if attrs["table_name"] != nil {
			s := attrs["table_name"].(string)
			table = &s
		}

		var column *string
		if attrs["column_name"] != nil {
			s := attrs["column_name"].(string)
			column = &s
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

		grantprivilege, err := dbopsClient.GetGrantPrivilege(ctx, accessType.(string), database, table, column, granteeUserName, granteeRoleName, clusterName)
		if err != nil {
			return err
		}

		if grantprivilege == nil {
			return fmt.Errorf("grantprivilege was not found")
		}

		if attrs["privilege_name"].(string) != grantprivilege.AccessType {
			return fmt.Errorf("expected privilege_name to be %q, was %q", grantprivilege.AccessType, attrs["privilege_name"].(string))
		}

		if !nilcompare.NilCompare(grantprivilege.DatabaseName, attrs["database_name"]) {
			return fmt.Errorf("wrong value for database attribute")
		}

		if !nilcompare.NilCompare(grantprivilege.TableName, attrs["table_name"]) {
			return fmt.Errorf("wrong value for table attribute")
		}

		if !nilcompare.NilCompare(grantprivilege.ColumnName, attrs["column_name"]) {
			return fmt.Errorf("wrong value for column attribute")
		}

		if !nilcompare.NilCompare(clusterName, attrs["cluster_name"]) {
			return fmt.Errorf("wrong value for cluster_name attribute")
		}

		if !nilcompare.NilCompare(grantprivilege.GranteeUserName, attrs["grantee_user_name"]) {
			return fmt.Errorf("wrong value for grantee_user_name attribute")
		}

		if !nilcompare.NilCompare(grantprivilege.GranteeRoleName, attrs["grantee_role_name"]) {
			return fmt.Errorf("wrong value for grantee_role_name attribute")
		}

		if grantprivilege.GrantOption != attrs["grant_option"].(bool) {
			return fmt.Errorf("wrong value for grant_option attribute")
		}

		return nil
	}

	tests := []runner.TestCase{
		// Single replica, Native
		{
			Name:     "Grant global privilege to role using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SHOW USERS").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant privilege to user on a database using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "CREATE TABLE").
				WithStringAttribute("database_name", "default").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant privilege to user on a table with grant option using Native protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("grant_option", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Single replica, HTTP
		{
			Name:     "Grant privilege on single column to role using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("column_name", "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant global privilege to user using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SHOW TABLES").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant privilege on database to user with grant option using HTTP protocol on a single replica",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "CREATE TABLE").
				WithStringAttribute("database_name", "default").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("grant_option", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, native
		{
			Name:     "Grant privilege on table to role using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant privilege on column to user using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("column_name", "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant global privilege to user with grant option using Native protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SHOW QUOTAS").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("grant_option", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Replicated storage, http
		{
			Name:     "Grant privilege on database to role using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "CREATE TABLE").
				WithStringAttribute("database_name", "default").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant privilege on table to user using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:     "Grant privilege on column to user with grant option using HTTP protocol on a cluster using replicated storage",
			ChEnv:    map[string]string{"CONFIGFILE": "config-replicated.xml"},
			Protocol: "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithStringAttribute("column_name", "name").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("grant_option", true).
				AddDependency(granteeUserResource.Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, native
		{
			Name:        "Grant global privilege to role using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("privilege_name", "SHOW ACCESS").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant privilege on database to user using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("privilege_name", "DROP TABLE").
				WithStringAttribute("database_name", "default").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant privilege on table to user with grant option using Native protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "native",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "databases").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("grant_option", true).
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		// Localfile storage, http
		{
			Name:        "Grant privilege on column to role using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("privilege_name", "SELECT").
				WithStringAttribute("database_name", "system").
				WithStringAttribute("table_name", "tables").
				WithStringAttribute("column_name", "name").
				WithResourceFieldReference("grantee_role_name", "clickhousedbops_role", granteeRoleName, "name").
				AddDependency(granteeRoleResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant global privilege to user using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("privilege_name", "SYSTEM RELOAD CONFIG").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				AddDependency(granteeUserResource.WithStringAttribute("cluster_name", clusterName).Build()).
				Build(),
			ResourceName:        resourceName,
			ResourceAddress:     fmt.Sprintf("%s.%s", resourceType, resourceName),
			CheckNotExistsFunc:  checkNotExistsFunc,
			CheckAttributesFunc: checkAttributesFunc,
		},
		{
			Name:        "Grant privilege on database to user with grant option using HTTP protocol on a cluster using localfile storage",
			ChEnv:       map[string]string{"CONFIGFILE": "config-localfile.xml"},
			ClusterName: &clusterName,
			Protocol:    "http",
			Resource: resourcebuilder.New(resourceType, resourceName).
				WithStringAttribute("cluster_name", clusterName).
				WithStringAttribute("privilege_name", "DROP VIEW").
				WithStringAttribute("database_name", "default").
				WithResourceFieldReference("grantee_user_name", "clickhousedbops_user", granteeUserName, "name").
				WithBoolAttribute("grant_option", true).
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
