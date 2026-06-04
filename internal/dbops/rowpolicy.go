package dbops

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type RowPolicy struct {
	Name             string
	Database         string
	Table            string
	ForOperations    []string // list of operations (e.g. "SELECT"). If empty, defaults to ["SELECT"]
	SelectFilter     string
	IsRestrictive    bool
	GranteeUserNames []string // list of user names
	GranteeRoleNames []string // list of role names
	GranteeAll       bool     // if true, applies to all
	GranteeAllExcept []string // list of roles/users to exclude from ALL
}

func (i *impl) CreateRowPolicy(ctx context.Context, rp RowPolicy, clusterName *string) (*RowPolicy, error) {
	toClause := i.buildGranteeClause(rp)
	if toClause == "" {
		return nil, errors.New("must specify at least one grantee: user, role, ALL, or ALL EXCEPT")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "CREATE ROW POLICY `%s`", rp.Name)

	if clusterName != nil && *clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER `%s`", *clusterName)
	}

	fmt.Fprintf(&sb, " ON `%s`.`%s`", rp.Database, rp.Table)

	// Only add FOR clause if operations are explicitly specified
	if len(rp.ForOperations) > 0 {
		for _, op := range rp.ForOperations {
			fmt.Fprintf(&sb, " FOR %s", op)
		}
	}
	fmt.Fprintf(&sb, " USING %s", rp.SelectFilter)

	if rp.IsRestrictive {
		sb.WriteString(" AS RESTRICTIVE")
	} else {
		sb.WriteString(" AS PERMISSIVE")
	}

	fmt.Fprintf(&sb, " TO %s", toClause)

	err := i.clickhouseClient.Exec(ctx, sb.String())
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	identifier := fmt.Sprintf("%s ON %s.%s", rp.Name, rp.Database, rp.Table)

	return retryWithBackoff(ctx, "row policy", identifier, func() (*RowPolicy, error) {
		return i.GetRowPolicy(ctx, &rp, clusterName)
	})
}

func (i *impl) buildGranteeClause(rp RowPolicy) string {
	if rp.GranteeAll {
		if len(rp.GranteeAllExcept) > 0 {
			var except []string
			for _, name := range rp.GranteeAllExcept {
				except = append(except, fmt.Sprintf("`%s`", name))
			}
			return fmt.Sprintf("ALL EXCEPT %s", strings.Join(except, ", "))
		}
		return "ALL"
	}

	var grantees []string

	for _, user := range rp.GranteeUserNames {
		grantees = append(grantees, fmt.Sprintf("`%s`", user))
	}

	for _, role := range rp.GranteeRoleNames {
		grantees = append(grantees, fmt.Sprintf("`%s`", role))
	}

	if len(grantees) > 0 {
		return strings.Join(grantees, ", ")
	}

	return ""
}
func (i *impl) GetRowPolicy(ctx context.Context, rp *RowPolicy, clusterName *string) (*RowPolicy, error) {
	where := []querybuilder.Where{
		querybuilder.WhereEquals("short_name", rp.Name),
		querybuilder.WhereEquals("database", rp.Database),
		querybuilder.WhereEquals("table", rp.Table),
	}

	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("short_name"),
			querybuilder.NewField("select_filter"),
			querybuilder.NewField("is_restrictive"),
		},
		"system.row_policies",
	).WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var result *RowPolicy
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		name, err := data.GetString("short_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'short_name' field")
		}

		// system.row_policies.select_filter is Nullable(String): it is NULL for a policy
		// created without a USING clause (e.g. a purely restrictive policy), so read it as
		// nullable and treat NULL as an empty filter.
		selectFilterPtr, err := data.GetNullableString("select_filter")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'select_filter' field")
		}
		selectFilter := ""
		if selectFilterPtr != nil {
			selectFilter = *selectFilterPtr
		}

		isRestrictive, err := data.GetBool("is_restrictive")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'is_restrictive' field")
		}

		result = &RowPolicy{
			Name:          name,
			Database:      rp.Database,
			Table:         rp.Table,
			SelectFilter:  selectFilter,
			IsRestrictive: isRestrictive,
		}

		// Populate grantees from input (they are write-once, so we keep them from the request)
		result.GranteeUserNames = rp.GranteeUserNames
		result.GranteeRoleNames = rp.GranteeRoleNames
		result.GranteeAll = rp.GranteeAll
		result.GranteeAllExcept = rp.GranteeAllExcept

		// Populate ForOperations from input (they are write-once, so we keep them from the request)
		result.ForOperations = rp.ForOperations

		// If SelectFilter is empty (couldn't be read), preserve the input value
		if result.SelectFilter == "" && rp.SelectFilter != "" {
			result.SelectFilter = rp.SelectFilter
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return result, nil
}

func (i *impl) UpdateRowPolicy(ctx context.Context, rp RowPolicy, clusterName *string) (*RowPolicy, error) {
	// Retrieve current row policy
	existing, err := i.GetRowPolicy(ctx, &rp, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to get existing row policy")
	}

	if existing == nil {
		return nil, errors.New("row policy not found")
	}

	builder := querybuilder.NewAlterRowPolicy(rp.Name, rp.Database, rp.Table)

	if clusterName != nil && *clusterName != "" {
		builder = builder.WithCluster(clusterName)
	}

	// Only include changes in the ALTER statement
	if rp.SelectFilter != existing.SelectFilter {
		builder = builder.SelectFilter(rp.SelectFilter)
	}

	if rp.IsRestrictive != existing.IsRestrictive {
		builder = builder.IsRestrictive(rp.IsRestrictive)
	}

	// Check if for operations have changed
	if !slices.Equal(rp.ForOperations, existing.ForOperations) {
		builder = builder.ForOperations(rp.ForOperations)
	}

	// Check if grantee specification has changed
	granteeChanged := !slices.Equal(rp.GranteeUserNames, existing.GranteeUserNames) ||
		!slices.Equal(rp.GranteeRoleNames, existing.GranteeRoleNames) ||
		rp.GranteeAll != existing.GranteeAll ||
		!slices.Equal(rp.GranteeAllExcept, existing.GranteeAllExcept)

	if granteeChanged {
		builder = builder.GranteeUserNames(rp.GranteeUserNames)
		builder = builder.GranteeRoleNames(rp.GranteeRoleNames)
		if rp.GranteeAll {
			builder = builder.GranteeAll(true)
		}
		builder = builder.GranteeAllExcept(rp.GranteeAllExcept)
	}

	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	identifier := fmt.Sprintf("%s ON %s.%s", rp.Name, rp.Database, rp.Table)

	return retryWithBackoff(ctx, "row policy", identifier, func() (*RowPolicy, error) {
		return i.GetRowPolicy(ctx, &rp, clusterName)
	})
}

func (i *impl) DeleteRowPolicy(ctx context.Context, name string, database string, table string, clusterName *string) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "DROP ROW POLICY IF EXISTS `%s`", name)

	if clusterName != nil && *clusterName != "" {
		fmt.Fprintf(&sb, " ON CLUSTER `%s`", *clusterName)
	}

	fmt.Fprintf(&sb, " ON `%s`.`%s`", database, table)

	err := i.clickhouseClient.Exec(ctx, sb.String())
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
