package revokeprivilege_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/testutils/runner"
)

const revokeResourceType = "clickhousedbops_revoke_privilege"

func TestRevokePrivilege_acceptance(t *testing.T) {
	check := func(ctx context.Context, client dbops.Client, clusterName *string, attrs map[string]any) error {
		partial := partialRevokeFromAttributes(attrs)
		found, err := client.GetPartialRevoke(ctx, &partial, clusterName)
		if err != nil {
			return err
		}
		if found == nil {
			return fmt.Errorf("partial privilege revoke was not found")
		}
		return nil
	}
	checkNotExists := func(ctx context.Context, client dbops.Client, clusterName *string, attrs map[string]string) (bool, error) {
		partial := partialRevokeFromStringAttributes(attrs)
		found, err := client.GetPartialRevoke(ctx, &partial, clusterName)
		return found != nil, err
	}

	tests := []runner.TestCase{
		{
			Name:     "Revoke a column privilege from a broader role grant",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "native",
			Resource: `
resource "clickhousedbops_role" "analyst" {
  name = "partial_revoke_analyst"
}

resource "clickhousedbops_grant_privilege" "reader" {
  privilege_name    = "SELECT"
  database_name     = "system"
  table_name        = "databases"
  grantee_role_name = clickhousedbops_role.analyst.name
}

resource "clickhousedbops_revoke_privilege" "test" {
  privilege_name    = "SELECT"
  database_name     = "system"
  table_name        = "databases"
  column_name       = "name"
  grantee_role_name = clickhousedbops_role.analyst.name
  depends_on        = [clickhousedbops_grant_privilege.reader]
}`,
			ResourceName:        "test",
			ResourceAddress:     revokeResourceType + ".test",
			CheckNotExistsFunc:  checkNotExists,
			CheckAttributesFunc: check,
		},
		{
			Name:     "Revoke only grant option from a user",
			ChEnv:    map[string]string{"CONFIGFILE": "config-single.xml"},
			Protocol: "http",
			Resource: `
resource "clickhousedbops_user" "reporter" {
  name                            = "partial_revoke_reporter"
  password_sha256_hash_wo         = sha256("test")
  password_sha256_hash_wo_version = 1
}

resource "clickhousedbops_grant_privilege" "reader" {
  privilege_name    = "SELECT"
  database_name     = "system"
  table_name        = "databases"
  grantee_user_name = clickhousedbops_user.reporter.name
  grant_option      = true
}

resource "clickhousedbops_revoke_privilege" "test" {
  privilege_name    = "SELECT"
  database_name     = "system"
  table_name        = "databases"
  column_name       = "name"
  grantee_user_name = clickhousedbops_user.reporter.name
  grant_option_only = true
  depends_on        = [clickhousedbops_grant_privilege.reader]
}`,
			ResourceName:        "test",
			ResourceAddress:     revokeResourceType + ".test",
			CheckNotExistsFunc:  checkNotExists,
			CheckAttributesFunc: check,
		},
	}

	runner.RunTests(t, tests)
}

func partialRevokeFromAttributes(attrs map[string]any) dbops.PartialRevoke {
	return dbops.PartialRevoke{
		AccessType:      attrs["privilege_name"].(string),
		DatabaseName:    anyStringPointer(attrs["database_name"]),
		TableName:       anyStringPointer(attrs["table_name"]),
		ColumnName:      anyStringPointer(attrs["column_name"]),
		AccessObject:    anyStringPointer(attrs["access_object"]),
		GranteeUserName: anyStringPointer(attrs["grantee_user_name"]),
		GranteeRoleName: anyStringPointer(attrs["grantee_role_name"]),
		GrantOptionOnly: attrs["grant_option_only"].(bool),
	}
}

func partialRevokeFromStringAttributes(attrs map[string]string) dbops.PartialRevoke {
	p := dbops.PartialRevoke{
		AccessType:      attrs["privilege_name"],
		DatabaseName:    stringPointer(attrs["database_name"]),
		TableName:       stringPointer(attrs["table_name"]),
		ColumnName:      stringPointer(attrs["column_name"]),
		AccessObject:    stringPointer(attrs["access_object"]),
		GranteeUserName: stringPointer(attrs["grantee_user_name"]),
		GranteeRoleName: stringPointer(attrs["grantee_role_name"]),
	}
	p.GrantOptionOnly = attrs["grant_option_only"] == "true"
	return p
}

func anyStringPointer(value any) *string {
	if value == nil {
		return nil
	}
	return new(value.(string))
}

func stringPointer(value string) *string {
	if value == "" {
		return nil
	}
	return new(value)
}
