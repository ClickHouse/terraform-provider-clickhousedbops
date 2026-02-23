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
	forOperations    []string // list of operations (e.g. "SELECT")
	selectFilter     *string
	isRestrictive    *bool
	granteeUserNames []string
	granteeRoleNames []string
	granteeAll       *bool
	granteeAllExcept []string
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

// ForOperations sets the operations for the row policy (e.g. "SELECT").
func (a *AlterRowPolicy) ForOperations(operations []string) *AlterRowPolicy {
	a.forOperations = operations
	return a
}

// SelectFilter sets the select filter for the row policy.
func (a *AlterRowPolicy) SelectFilter(filter string) *AlterRowPolicy {
	a.selectFilter = &filter
	return a
}

// IsRestrictive sets whether the policy is restrictive or permissive.
func (a *AlterRowPolicy) IsRestrictive(restrictive bool) *AlterRowPolicy {
	a.isRestrictive = &restrictive
	return a
}

// GranteeUserNames sets the user names for the TO clause.
func (a *AlterRowPolicy) GranteeUserNames(users []string) *AlterRowPolicy {
	a.granteeUserNames = users
	return a
}

// GranteeRoleNames sets the role names for the TO clause.
func (a *AlterRowPolicy) GranteeRoleNames(roles []string) *AlterRowPolicy {
	a.granteeRoleNames = roles
	return a
}

// GranteeAll sets whether the policy applies to all users/roles.
func (a *AlterRowPolicy) GranteeAll(all bool) *AlterRowPolicy {
	a.granteeAll = &all
	return a
}

// GranteeAllExcept sets the exclusion list for ALL EXCEPT.
func (a *AlterRowPolicy) GranteeAllExcept(except []string) *AlterRowPolicy {
	a.granteeAllExcept = except
	return a
}

// Build generates the ALTER ROW POLICY SQL statement.
func (a *AlterRowPolicy) Build() (string, error) {
	var sb strings.Builder

	fmt.Fprintf(&sb, "ALTER ROW POLICY `%s`", a.name)

	if a.clusterName != nil && *a.clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER `%s`", *a.clusterName)
	}

	fmt.Fprintf(&sb, " ON `%s`.`%s`", a.database, a.table)

	// At least one modification is required
	hasChanges := false

	// Handle FOR <operations> clause (currently SELECT, but future-proof for other operations)
	if len(a.forOperations) > 0 {
		for _, op := range a.forOperations {
			fmt.Fprintf(&sb, " FOR %s", op)
		}
		hasChanges = true
	}

	// Handle AS PERMISSIVE/RESTRICTIVE and USING clauses (independent of FOR operations)
	if a.selectFilter != nil || a.isRestrictive != nil {
		// Only add FOR SELECT if no FOR operations were already added and we're modifying SELECT-related clauses
		if len(a.forOperations) == 0 && (a.selectFilter != nil || a.isRestrictive != nil) {
			// Don't add FOR SELECT - let user specify it explicitly if needed
		}
		hasChanges = true

		if a.isRestrictive != nil {
			if *a.isRestrictive {
				sb.WriteString(" AS RESTRICTIVE")
			} else {
				sb.WriteString(" AS PERMISSIVE")
			}
		}

		if a.selectFilter != nil {
			fmt.Fprintf(&sb, " USING %s", *a.selectFilter)
		}
	}

	// Check if grantee specification has been set
	hasGranteeSpec := len(a.granteeUserNames) > 0 || len(a.granteeRoleNames) > 0 ||
		(a.granteeAll != nil && *a.granteeAll) || len(a.granteeAllExcept) > 0

	if hasGranteeSpec {
		fmt.Fprintf(&sb, " TO %s", a.buildGranteeClause())
		hasChanges = true
	}

	if !hasChanges {
		return "", fmt.Errorf("at least one change must be specified for ALTER ROW POLICY")
	}

	return sb.String(), nil
}

// buildGranteeClause builds the TO clause for grantee specification.
func (a *AlterRowPolicy) buildGranteeClause() string {
	if a.granteeAll != nil && *a.granteeAll {
		if len(a.granteeAllExcept) > 0 {
			var except []string
			for _, name := range a.granteeAllExcept {
				except = append(except, fmt.Sprintf("`%s`", name))
			}
			return fmt.Sprintf("ALL EXCEPT %s", strings.Join(except, ", "))
		}
		return "ALL"
	}

	var names []string
	for _, name := range a.granteeUserNames {
		names = append(names, fmt.Sprintf("`%s`", name))
	}
	for _, name := range a.granteeRoleNames {
		names = append(names, fmt.Sprintf("`%s`", name))
	}
	return strings.Join(names, ", ")
}
