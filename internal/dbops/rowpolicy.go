package dbops

import (
	"context"
	"fmt"

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
	sql, err := querybuilder.NewCreateRowPolicy(rp.Name, rp.Database, rp.Table).
		WithCluster(clusterName).
		ForOperations(rp.ForOperations).
		SelectFilter(rp.SelectFilter).
		IsRestrictive(rp.IsRestrictive).
		GranteeUserNames(rp.GranteeUserNames).
		GranteeRoleNames(rp.GranteeRoleNames).
		GranteeAll(rp.GranteeAll).
		GranteeAllExcept(rp.GranteeAllExcept).
		Build()
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
	// select_filter and is_restrictive force replacement (see the resource schema), so an Update
	// only ever changes grantees or for_operations. GetRowPolicy doesn't read those back from the
	// DB (it echoes the request), so instead of diffing we set the desired grantees and operations
	// on the policy directly. ALTER ROW POLICY ... TO ... replaces the grantee set.
	builder := querybuilder.NewAlterRowPolicy(rp.Name, rp.Database, rp.Table)

	if clusterName != nil && *clusterName != "" {
		builder = builder.WithCluster(clusterName)
	}

	if len(rp.ForOperations) > 0 {
		builder = builder.ForOperations(rp.ForOperations)
	}

	builder = builder.GranteeUserNames(rp.GranteeUserNames)
	builder = builder.GranteeRoleNames(rp.GranteeRoleNames)
	if rp.GranteeAll {
		builder = builder.GranteeAll(true)
	}
	builder = builder.GranteeAllExcept(rp.GranteeAllExcept)

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
	sql, err := querybuilder.NewDropRowPolicy(name, database, table).
		WithCluster(clusterName).
		IfExists(true).
		Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
