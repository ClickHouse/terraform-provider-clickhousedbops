package dbops

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type RowPolicy struct {
	ID               string
	Name             string
	Database         string
	Table            string
	SelectFilter     string
	IsRestrictive    bool
	GranteeNames     []string // list of usernames and roles
	GranteeAll       bool     // if true, applies to all
	GranteeAllExcept []string // list of roles/users to exclude from ALL
}

func (i *impl) CreateRowPolicy(ctx context.Context, rp RowPolicy, clusterName *string) (*RowPolicy, error) {
	sql, err := querybuilder.NewCreateRowPolicy(rp.Name, rp.Database, rp.Table).
		WithCluster(clusterName).
		SelectFilter(rp.SelectFilter).
		IsRestrictive(rp.IsRestrictive).
		GranteeNames(rp.GranteeNames).
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
	return i.getRowPolicyWhere(ctx, []querybuilder.Where{
		querybuilder.WhereEquals("short_name", rp.Name),
		querybuilder.WhereEquals("database", rp.Database),
		querybuilder.WhereEquals("table", rp.Table),
	}, clusterName)
}

func (i *impl) GetRowPolicyByID(ctx context.Context, id string, clusterName *string) (*RowPolicy, error) {
	return i.getRowPolicyWhere(ctx, []querybuilder.Where{
		querybuilder.WhereEquals("id", id),
	}, clusterName)
}

func (i *impl) getRowPolicyWhere(ctx context.Context, where []querybuilder.Where, clusterName *string) (*RowPolicy, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("id").ToString(),
			querybuilder.NewField("short_name"),
			querybuilder.NewField("database"),
			querybuilder.NewField("table"),
			querybuilder.NewField("select_filter"),
			querybuilder.NewField("is_restrictive"),
			querybuilder.NewField("apply_to_all"),
			// apply_to_list/apply_to_except are Array(String); flatten to a newline-joined scalar so
			// they read back as a plain String on both the native and http transports (the http
			// JSONCompactStrings parser has no Array(String) case).
			querybuilder.NewRawField("arrayStringConcat(apply_to_list, '\\n')", "apply_to_list"),
			querybuilder.NewRawField("arrayStringConcat(apply_to_except, '\\n')", "apply_to_except"),
		},
		"system.row_policies",
	).WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var result *RowPolicy
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		id, err := data.GetString("id")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'id' field")
		}

		name, err := data.GetString("short_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'short_name' field")
		}

		database, err := data.GetString("database")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'database' field")
		}

		table, err := data.GetString("table")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'table' field")
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

		applyToAll, err := data.GetBool("apply_to_all")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'apply_to_all' field")
		}

		applyToList, err := data.GetString("apply_to_list")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'apply_to_list' field")
		}

		applyToExcept, err := data.GetString("apply_to_except")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'apply_to_except' field")
		}

		result = &RowPolicy{
			ID:               id,
			Name:             name,
			Database:         database,
			Table:            table,
			SelectFilter:     selectFilter,
			IsRestrictive:    isRestrictive,
			GranteeAll:       applyToAll,
			GranteeNames:     splitNonEmpty(applyToList),
			GranteeAllExcept: splitNonEmpty(applyToExcept),
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return result, nil
}

// splitNonEmpty splits a newline-joined string into its parts, returning nil for the empty string
// (so an empty grantee list maps to a null Terraform set rather than [""]).
func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

// UpdateRowPolicy re-asserts the full desired policy (name, filter, restrictiveness and grantees) in a single ALTER ROW POLICY.
func (i *impl) UpdateRowPolicy(ctx context.Context, rp RowPolicy, clusterName *string) (*RowPolicy, error) {
	existing, err := i.GetRowPolicyByID(ctx, rp.ID, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to get existing row policy")
	}
	if existing == nil {
		return nil, errors.Errorf("row policy with id %q not found", rp.ID)
	}

	builder := querybuilder.NewAlterRowPolicy(existing.Name, existing.Database, existing.Table).
		RenameTo(rp.Name).
		SelectFilter(rp.SelectFilter).
		IsRestrictive(rp.IsRestrictive).
		GranteeNames(rp.GranteeNames).
		GranteeAllExcept(rp.GranteeAllExcept)

	if clusterName != nil && *clusterName != "" {
		builder = builder.WithCluster(clusterName)
	}

	if rp.GranteeAll {
		builder = builder.GranteeAll(true)
	}

	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return retryWithBackoff(ctx, "row policy", rp.ID, func() (*RowPolicy, error) {
		return i.GetRowPolicyByID(ctx, rp.ID, clusterName)
	})
}

func (i *impl) DeleteRowPolicy(ctx context.Context, id string, clusterName *string) error {
	rp, err := i.GetRowPolicyByID(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error getting row policy")
	}

	if rp == nil {
		// Already gone, nothing to do.
		return nil
	}

	sql, err := querybuilder.NewDropRowPolicy(rp.Name, rp.Database, rp.Table).
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
