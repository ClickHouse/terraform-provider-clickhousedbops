package grantprivilege

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

func overlaps(current GrantPrivilege, existing dbops.GrantPrivilege) bool {
	// AccessType: existing must be the same privilege, or a group that contains current.
	if !slices.Contains(AllDescendants(parsedGrants().Groups, existing.AccessType), current.Privilege.ValueString()) {
		return false
	}

	// A grant that needs grant option is not covered by one lacking it.
	if current.GrantOption.ValueBool() && !existing.GrantOption {
		return false
	}

	attrs, _, ok := scopeAttributesFor(current.Privilege.ValueString())
	if !ok {
		return false
	}

	if attrs.database && !checkWildcardOverlaps(current.Database, existing.DatabaseName) {
		return false
	}
	if attrs.table && !checkWildcardOverlaps(current.Table, existing.TableName) {
		return false
	}
	if attrs.column && !checkWildcardOverlaps(current.Column, existing.ColumnName) {
		return false
	}

	return true
}

func explainOverlap(current GrantPrivilege, existing dbops.GrantPrivilege) string {
	// Prepare human-readable explanation of the overlap.
	var row string
	if current.Privilege.ValueString() != existing.AccessType {
		row = fmt.Sprintf("- Broader privilege %q (which includes %q) is already granted", existing.AccessType, current.Privilege.ValueString())
	} else {
		row = fmt.Sprintf("- Privilege %q is already granted", existing.AccessType)
	}

	// The target description depends on the grant's scope dimension.
	switch {
	case existing.ColumnName != nil:
		row = fmt.Sprintf("%s on column %q of table %q in the %q database", row, *existing.ColumnName, *existing.TableName, *existing.DatabaseName)
	case existing.TableName != nil:
		row = fmt.Sprintf("%s on table %q in the %q database", row, *existing.TableName, *existing.DatabaseName)
	case existing.DatabaseName != nil:
		row = fmt.Sprintf("%s on all tables in the %q database", row, *existing.DatabaseName)
	default:
		row = fmt.Sprintf("%s on all", row)
	}

	if existing.GranteeUserName != nil {
		row = fmt.Sprintf("%s to user %q", row, *existing.GranteeUserName)
	}

	if existing.GranteeRoleName != nil {
		row = fmt.Sprintf("%s to role %q", row, *existing.GranteeRoleName)
	}

	if !current.GrantOption.IsUnknown() && current.GrantOption.ValueBool() != existing.GrantOption {
		if existing.GrantOption {
			row = fmt.Sprintf("%s with grant option", row)
		} else {
			row = fmt.Sprintf("%s without grant option", row)
		}
	}

	return row
}

func checkWildcardOverlaps(current types.String, existing *string) bool {
	// existing is unrestricted on this level: it covers anything (incl. current = all).
	if existing == nil {
		return true
	}
	// existing is specific but current is unrestricted (all): not covered.
	if current.IsNull() {
		return false
	}
	if current.ValueString() == *existing {
		return true
	}
	// existing is a prefix wildcard: it covers current when current starts with the prefix.
	if strings.HasSuffix(*existing, "*") {
		return strings.HasPrefix(current.ValueString(), strings.TrimSuffix(*existing, "*"))
	}
	// existing is an exact value different from current.
	return false
}
