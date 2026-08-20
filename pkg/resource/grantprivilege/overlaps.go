package grantprivilege

import (
	"fmt"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
)

// overlaps reports whether an already-granted privilege covers the one in current.
func overlaps(current GrantPrivilege, existing dbops.GrantPrivilege) bool {
	return grants.Covers(existing.AsGrant(), current.asGrant())
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
	case existing.AccessObject != nil:
		row = fmt.Sprintf("%s on %q", row, *existing.AccessObject)
		if existing.AccessObjectFilter != nil {
			row = fmt.Sprintf("%s with filter %q", row, *existing.AccessObjectFilter)
		}
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
