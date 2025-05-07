package grantprivilege

import (
	"fmt"
	"strings"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
)

func overlaps(current GrantPrivilege, existing dbops.GrantPrivilege) bool {
	// AccessType
	{
		if current.Privilege.ValueString() != existing.AccessType {
			// Check if existing privilege is a group containing current one.

			groups := parseGrants().Groups

			if members := groups[existing.AccessType]; members != nil {
				found := false
				for _, m := range members {
					if m == current.Privilege.ValueString() {
						found = true
						break
					}
				}

				if !found {
					// Existing is a group, but current is not a member of the group
					return false
				}
			} else {
				// existing is not a group
				return false
			}
		}
	}

	// DatabaseName
	{
		if !current.Database.IsNull() && existing.DatabaseName != nil && current.Database.ValueString() != *existing.DatabaseName {
			// DatabaseName is different, but it can still be overlapping if using wildcards.
			if strings.HasSuffix(current.Database.ValueString(), "*") {
				if strings.HasSuffix(*existing.DatabaseName, "*") {
					// Both DatabaseNames end with a wildcard.
					if !strings.HasPrefix(current.Database.ValueString(), strings.TrimSuffix(*existing.DatabaseName, "*")) {
						return false
					}
				} else {
					// Current ends with a wildcard, existing does not.
					return false
				}
			} else {
				if strings.HasSuffix(*existing.DatabaseName, "*") {
				} else {
					// Both DatabaseNames do not have wildcard and are different.
					return false
				}
			}
		} else if current.Database.IsNull() && existing.DatabaseName != nil {
			return false
		}
	}

	// TableName
	{
		if !current.Table.IsNull() && existing.TableName != nil && current.Table.ValueString() != *existing.TableName {
			// TableName is different, but it can still be overlapping if using wildcards.
			if strings.HasSuffix(current.Table.ValueString(), "*") {
				if strings.HasSuffix(*existing.TableName, "*") {
					// Both TableNames end with a wildcard.
					if !strings.HasPrefix(current.Table.ValueString(), strings.TrimSuffix(*existing.TableName, "*")) {
						return false
					}
				} else {
					// Current ends with a wildcard, existing does not.
					return false
				}
			} else {
				if strings.HasSuffix(*existing.TableName, "*") {
				} else {
					// Both TableNames do not have wildcard and are different.
					return false
				}
			}
		} else if current.Table.IsNull() && existing.TableName != nil {
			return false
		}
	}

	// ColumnName
	{
		if !current.Column.IsNull() && existing.ColumnName != nil {
			if current.Column.ValueString() != *existing.ColumnName {
				return false
			}
		} else if current.Column.IsNull() && existing.ColumnName != nil {
			// current is for all columns, existing if for specific column
			return false
		}
	}

	// GranteeUserName
	{
		if !current.GranteeUserName.IsNull() && existing.GranteeUserName != nil && current.GranteeUserName.ValueString() != *existing.GranteeUserName {
			return false
		} else if !current.GranteeUserName.IsNull() && existing.GranteeUserName == nil {
			return false
		} else if current.GranteeUserName.IsNull() && existing.GranteeUserName != nil {
			return false
		}
	}

	// GranteeRoleName
	{
		if !current.GranteeRoleName.IsNull() && existing.GranteeRoleName != nil && current.GranteeRoleName.ValueString() != *existing.GranteeRoleName {
			return false
		} else if !current.GranteeRoleName.IsNull() && existing.GranteeRoleName == nil {
			return false
		} else if current.GranteeRoleName.IsNull() && existing.GranteeRoleName != nil {
			return false
		}
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

	if existing.TableName != nil {
		row = fmt.Sprintf("%s on table %q", row, *existing.TableName)
	} else {
		row = fmt.Sprintf("%s on all tables", row)
	}

	if existing.DatabaseName != nil {
		row = fmt.Sprintf("%s in the %q database", row, *existing.DatabaseName)
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
