package querybuilder

import (
	"fmt"
	"strings"
)

// CreateRowPolicy builds CREATE ROW POLICY statements. Identifiers go through backtick() so they
// are escaped consistently with the ALTER ROW POLICY builder.
type CreateRowPolicy struct {
	name             string
	database         string
	table            string
	clusterName      *string
	forOperations    []string
	selectFilter     string
	isRestrictive    bool
	granteeUserNames []string
	granteeRoleNames []string
	granteeAll       bool
	granteeAllExcept []string
}

// NewCreateRowPolicy creates a new CreateRowPolicy builder.
func NewCreateRowPolicy(name string, database string, table string) *CreateRowPolicy {
	return &CreateRowPolicy{
		name:     name,
		database: database,
		table:    table,
	}
}

// WithCluster sets the cluster name for the CREATE statement.
func (c *CreateRowPolicy) WithCluster(clusterName *string) *CreateRowPolicy {
	c.clusterName = clusterName
	return c
}

// ForOperations sets the operations for the row policy (e.g. "SELECT").
func (c *CreateRowPolicy) ForOperations(operations []string) *CreateRowPolicy {
	c.forOperations = operations
	return c
}

// SelectFilter sets the USING filter for the row policy.
func (c *CreateRowPolicy) SelectFilter(filter string) *CreateRowPolicy {
	c.selectFilter = filter
	return c
}

// IsRestrictive sets whether the policy is restrictive or permissive.
func (c *CreateRowPolicy) IsRestrictive(restrictive bool) *CreateRowPolicy {
	c.isRestrictive = restrictive
	return c
}

// GranteeUserNames sets the user names for the TO clause.
func (c *CreateRowPolicy) GranteeUserNames(users []string) *CreateRowPolicy {
	c.granteeUserNames = users
	return c
}

// GranteeRoleNames sets the role names for the TO clause.
func (c *CreateRowPolicy) GranteeRoleNames(roles []string) *CreateRowPolicy {
	c.granteeRoleNames = roles
	return c
}

// GranteeAll sets whether the policy applies to all users/roles.
func (c *CreateRowPolicy) GranteeAll(all bool) *CreateRowPolicy {
	c.granteeAll = all
	return c
}

// GranteeAllExcept sets the exclusion list for ALL EXCEPT.
func (c *CreateRowPolicy) GranteeAllExcept(except []string) *CreateRowPolicy {
	c.granteeAllExcept = except
	return c
}

// Build generates the CREATE ROW POLICY SQL statement.
func (c *CreateRowPolicy) Build() (string, error) {
	if c.name == "" {
		return "", fmt.Errorf("row policy name cannot be empty")
	}
	if c.database == "" || c.table == "" {
		return "", fmt.Errorf("database and table are required for row policies")
	}
	if c.selectFilter == "" {
		return "", fmt.Errorf("select filter is required")
	}

	grantees := c.buildGranteeClause()
	if grantees == "" {
		return "", fmt.Errorf("must specify at least one grantee: user, role, ALL, or ALL EXCEPT")
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "CREATE ROW POLICY %s", backtick(c.name))

	if c.clusterName != nil && *c.clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER %s", backtick(*c.clusterName))
	}

	fmt.Fprintf(&sb, " ON %s.%s", backtick(c.database), backtick(c.table))

	for _, op := range c.forOperations {
		fmt.Fprintf(&sb, " FOR %s", op)
	}

	fmt.Fprintf(&sb, " USING %s", c.selectFilter)

	if c.isRestrictive {
		sb.WriteString(" AS RESTRICTIVE")
	} else {
		sb.WriteString(" AS PERMISSIVE")
	}

	fmt.Fprintf(&sb, " TO %s", grantees)

	return sb.String(), nil
}

// DropRowPolicy builds DROP ROW POLICY statements.
type DropRowPolicy struct {
	name        string
	database    string
	table       string
	clusterName *string
	ifExists    bool
}

// NewDropRowPolicy creates a new DropRowPolicy builder.
func NewDropRowPolicy(name string, database string, table string) *DropRowPolicy {
	return &DropRowPolicy{
		name:     name,
		database: database,
		table:    table,
	}
}

// WithCluster sets the cluster name for the DROP statement.
func (d *DropRowPolicy) WithCluster(clusterName *string) *DropRowPolicy {
	d.clusterName = clusterName
	return d
}

// IfExists adds the IF EXISTS clause.
func (d *DropRowPolicy) IfExists(v bool) *DropRowPolicy {
	d.ifExists = v
	return d
}

// Build generates the DROP ROW POLICY SQL statement.
func (d *DropRowPolicy) Build() (string, error) {
	if d.name == "" {
		return "", fmt.Errorf("row policy name cannot be empty")
	}
	if d.database == "" || d.table == "" {
		return "", fmt.Errorf("database and table are required for row policies")
	}

	var sb strings.Builder

	sb.WriteString("DROP ROW POLICY")
	if d.ifExists {
		sb.WriteString(" IF EXISTS")
	}
	fmt.Fprintf(&sb, " %s", backtick(d.name))

	if d.clusterName != nil && *d.clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER %s", backtick(*d.clusterName))
	}

	fmt.Fprintf(&sb, " ON %s.%s", backtick(d.database), backtick(d.table))

	return sb.String(), nil
}

// buildGranteeClause builds the TO clause for grantee specification.
func (c *CreateRowPolicy) buildGranteeClause() string {
	if c.granteeAll {
		if len(c.granteeAllExcept) > 0 {
			return fmt.Sprintf("ALL EXCEPT %s", strings.Join(backtickAll(c.granteeAllExcept), ", "))
		}
		return "ALL"
	}

	names := make([]string, 0, len(c.granteeUserNames)+len(c.granteeRoleNames))
	names = append(names, backtickAll(c.granteeUserNames)...)
	names = append(names, backtickAll(c.granteeRoleNames)...)
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, ", ")
}
