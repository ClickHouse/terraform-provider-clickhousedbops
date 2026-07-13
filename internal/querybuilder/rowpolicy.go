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
	selectFilter     string
	isRestrictive    bool
	granteeNames     []string
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

// GranteeNames sets the user and role names for the TO clause.
func (c *CreateRowPolicy) GranteeNames(users []string) *CreateRowPolicy {
	c.granteeNames = users
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

	grantees := granteeClause(c.granteeNames, c.granteeAll, c.granteeAllExcept)
	if grantees == "" {
		return "", fmt.Errorf("must specify at least one grantee: user, role, ALL, or ALL EXCEPT")
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "CREATE ROW POLICY %s", backtick(c.name))

	if c.clusterName != nil && *c.clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER %s", backtick(*c.clusterName))
	}

	fmt.Fprintf(&sb, " ON %s.%s", backtick(c.database), backtick(c.table))

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
