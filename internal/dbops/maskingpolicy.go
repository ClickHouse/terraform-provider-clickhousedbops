package dbops

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// ColumnMask is a single `column = expression` assignment. Re-exported so the resource layer does
// not need to import the querybuilder package.
type ColumnMask = querybuilder.ColumnMask

// MaskingPolicy is a ClickHouse Cloud masking policy: it rewrites the listed columns of a table
// for the grantees, optionally only for rows matching Where.
type MaskingPolicy struct {
	ID               string
	Name             string
	Database         string
	Table            string
	Masks            []ColumnMask
	AssignmentsHash  string // Hash of the assignments to detect server-side drift.
	Where            string
	GranteeNames     []string
	GranteeAll       bool
	GranteeAllExcept []string
	Priority         *int64
}

func (m *MaskingPolicy) identifier() string {
	return fmt.Sprintf("%s ON %s.%s", m.Name, m.Database, m.Table)
}

func whereOrNil(s string) *string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

// CreateMaskingPolicy creates the policy with a plain CREATE so that an existing policy of the same
// name on the same table surfaces an "already exists" error instead of being silently overwritten;
// the resource layer turns that error into an import hint.
func (i *impl) CreateMaskingPolicy(ctx context.Context, mp MaskingPolicy) (*MaskingPolicy, error) {
	sql, err := querybuilder.NewCreateMaskingPolicy(mp.Name, mp.Database, mp.Table, mp.Masks).
		WithWhere(whereOrNil(mp.Where)).
		GranteeNames(mp.GranteeNames).
		GranteeAll(mp.GranteeAll).
		GranteeAllExcept(mp.GranteeAllExcept).
		WithPriority(mp.Priority).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	if err := i.clickhouseClient.Exec(ctx, sql); err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return retryWithBackoff(ctx, "masking policy", mp.identifier(), func() (*MaskingPolicy, error) {
		return i.GetMaskingPolicy(ctx, &mp)
	}, i.readAfterWriteTimeoutArgs()...)
}

// UpdateMaskingPolicy re-asserts the full desired definition with a single ALTER MASKING POLICY.
func (i *impl) UpdateMaskingPolicy(ctx context.Context, mp MaskingPolicy) (*MaskingPolicy, error) {
	existing, err := i.GetMaskingPolicyByID(ctx, mp.ID)
	if err != nil {
		return nil, errors.WithMessage(err, "unable to get existing masking policy")
	}
	if existing == nil {
		return nil, errors.Errorf("masking policy with id %q not found", mp.ID)
	}

	sql, err := querybuilder.NewAlterMaskingPolicy(existing.Name, existing.Database, existing.Table, mp.Masks).
		RenameTo(mp.Name).
		WithWhere(whereOrNil(mp.Where)).
		GranteeNames(mp.GranteeNames).
		GranteeAll(mp.GranteeAll).
		GranteeAllExcept(mp.GranteeAllExcept).
		WithPriority(mp.Priority).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	if err := i.clickhouseClient.Exec(ctx, sql); err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return retryWithBackoff(ctx, "masking policy", mp.identifier(), func() (*MaskingPolicy, error) {
		return i.GetMaskingPolicyByID(ctx, mp.ID)
	}, i.readAfterWriteTimeoutArgs()...)
}

func (i *impl) GetMaskingPolicy(ctx context.Context, mp *MaskingPolicy) (*MaskingPolicy, error) {
	return i.getMaskingPolicyWhere(ctx, []querybuilder.Where{
		querybuilder.WhereEquals("short_name", mp.Name),
		querybuilder.WhereEquals("database", mp.Database),
		querybuilder.WhereEquals("table", mp.Table),
	})
}

func (i *impl) GetMaskingPolicyByID(ctx context.Context, id string) (*MaskingPolicy, error) {
	return i.getMaskingPolicyWhere(ctx, []querybuilder.Where{
		querybuilder.WhereEquals("id", id),
	})
}

func (i *impl) getMaskingPolicyWhere(ctx context.Context, where []querybuilder.Where) (*MaskingPolicy, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("id").ToString(),
			querybuilder.NewField("short_name"),
			querybuilder.NewField("database"),
			querybuilder.NewField("table"),
			querybuilder.NewRawField("toString(cityHash64(ifNull(update_assignments, '')))", "update_assignments_hash"),
			querybuilder.NewField("where_condition"),
			querybuilder.NewRawField("toString(priority)", "priority"),
			querybuilder.NewField("apply_to_all"),
			querybuilder.NewRawField("arrayStringConcat(apply_to_list, '\\n')", "apply_to_list"),
			querybuilder.NewRawField("arrayStringConcat(apply_to_except, '\\n')", "apply_to_except"),
		},
		"system.masking_policies",
	).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var result *MaskingPolicy
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

		assignmentsHash, err := data.GetString("update_assignments_hash")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'update_assignments_hash' field")
		}

		wherePtr, err := data.GetNullableString("where_condition")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'where_condition' field")
		}
		whereCondition := ""
		if wherePtr != nil {
			whereCondition = *wherePtr
		}

		priorityStr, err := data.GetString("priority")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'priority' field")
		}
		priority, err := strconv.ParseInt(priorityStr, 10, 64)
		if err != nil {
			return errors.WithMessage(err, "error parsing 'priority' field")
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

		result = &MaskingPolicy{
			ID:               id,
			Name:             name,
			Database:         database,
			Table:            table,
			AssignmentsHash:  assignmentsHash,
			Where:            whereCondition,
			GranteeNames:     splitNonEmpty(applyToList),
			GranteeAll:       applyToAll,
			GranteeAllExcept: splitNonEmpty(applyToExcept),
			Priority:         &priority,
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return result, nil
}

func (i *impl) DeleteMaskingPolicy(ctx context.Context, id string) error {
	mp, err := i.GetMaskingPolicyByID(ctx, id)
	if err != nil {
		return errors.WithMessage(err, "error getting masking policy")
	}

	if mp == nil {
		// Already gone, nothing to do.
		return nil
	}

	sql, err := querybuilder.NewDropMaskingPolicy(mp.Name, mp.Database, mp.Table).IfExists(true).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	if err := i.clickhouseClient.Exec(ctx, sql); err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
