package querybuilder

import (
	"fmt"
	"strings"
)

// AlterRowPolicy is a query builder for ALTER ROW POLICY statements.
type AlterRowPolicy struct {
	name             string
	database         string
	table            string
	clusterName      *string
	renameTo         *string
	selectFilter     *string
	isRestrictive    *bool
	granteeNames     []string
	granteeAll       *bool
	granteeAllExcept []string

	hasChanges bool
}

// NewAlterRowPolicy creates a new AlterRowPolicy builder.
func NewAlterRowPolicy(name string, database string, table string) *AlterRowPolicy {
	return &AlterRowPolicy{
		name:     name,
		database: database,
		table:    table,
	}
}

// WithCluster sets the cluster name for the ALTER statement.
func (a *AlterRowPolicy) WithCluster(clusterName *string) *AlterRowPolicy {
	a.clusterName = clusterName
	return a
}

// RenameTo renames the row policy. The rename is emitted only when the new name differs from the current one.
func (a *AlterRowPolicy) RenameTo(newName string) *AlterRowPolicy {
	a.renameTo = &newName
	a.hasChanges = true
	return a
}

// SelectFilter sets the select filter for the row policy.
func (a *AlterRowPolicy) SelectFilter(filter string) *AlterRowPolicy {
	a.selectFilter = &filter
	a.hasChanges = true
	return a
}

// IsRestrictive sets whether the policy is restrictive or permissive.
func (a *AlterRowPolicy) IsRestrictive(restrictive bool) *AlterRowPolicy {
	a.isRestrictive = &restrictive
	a.hasChanges = true
	return a
}

// GranteeNames sets the user and role names for the TO clause.
func (a *AlterRowPolicy) GranteeNames(users []string) *AlterRowPolicy {
	a.granteeNames = users
	a.hasChanges = true
	return a
}

// GranteeAll sets whether the policy applies to all users/roles.
func (a *AlterRowPolicy) GranteeAll(all bool) *AlterRowPolicy {
	a.granteeAll = &all
	a.hasChanges = true
	return a
}

// GranteeAllExcept sets the exclusion list for ALL EXCEPT.
func (a *AlterRowPolicy) GranteeAllExcept(except []string) *AlterRowPolicy {
	a.granteeAllExcept = except
	a.hasChanges = true
	return a
}

// Build generates the ALTER ROW POLICY SQL statement.
func (a *AlterRowPolicy) Build() (string, error) {
	if !a.hasChanges {
		return "", fmt.Errorf("at least one change must be specified for ALTER ROW POLICY")
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "ALTER ROW POLICY %s", backtick(a.name))

	if a.clusterName != nil && *a.clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER %s", backtick(*a.clusterName))
	}

	fmt.Fprintf(&sb, " ON %s.%s", backtick(a.database), backtick(a.table))

	if a.renameTo != nil && *a.renameTo != a.name {
		fmt.Fprintf(&sb, " RENAME TO %s", backtick(*a.renameTo))
	}

	if a.isRestrictive != nil {
		if *a.isRestrictive {
			sb.WriteString(" AS RESTRICTIVE")
		} else {
			sb.WriteString(" AS PERMISSIVE")
		}
	}

	if a.selectFilter != nil {
		sb.WriteString(" USING ")
		sb.WriteString(*a.selectFilter)
	}

	// Check if grantee specification has been set
	hasGranteeSpec := len(a.granteeNames) > 0 || (a.granteeAll != nil && *a.granteeAll) || len(a.granteeAllExcept) > 0

	if hasGranteeSpec {
		fmt.Fprintf(&sb, " TO %s", granteeClause(a.granteeNames, a.granteeAll != nil && *a.granteeAll, a.granteeAllExcept))
	}

	return sb.String(), nil
}
