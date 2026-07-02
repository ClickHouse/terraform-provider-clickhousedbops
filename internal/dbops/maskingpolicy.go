package dbops

import (
	"context"
	"fmt"
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
	Name             string
	Database         string
	Table            string
	Masks            []ColumnMask
	Where            string
	GranteeUserNames []string
	GranteeRoleNames []string
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

// CreateMaskingPolicy applies the policy with CREATE OR REPLACE so it is idempotent and so updates
// replace the previous definition atomically (no unmasked window).
func (i *impl) CreateMaskingPolicy(ctx context.Context, mp MaskingPolicy) (*MaskingPolicy, error) {
	sql, err := querybuilder.NewCreateMaskingPolicy(mp.Name, mp.Database, mp.Table, mp.Masks).
		OrReplace(true).
		WithWhere(whereOrNil(mp.Where)).
		WithGrantees(mp.GranteeUserNames, mp.GranteeRoleNames, mp.GranteeAll, mp.GranteeAllExcept).
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

// GetMaskingPolicy confirms the policy exists by looking it up in system.masking_policies on
// (short_name, database, table), the same key the row-policy resource uses against system.row_policies.
// SHOW MASKING POLICIES only exposes a compound `name ON db.table`, so matching by name prefix could
// pick the wrong policy when the same short name is reused across tables; the system table keys on
// the full tuple. Masking expressions are not introspectable back into the config shape, so the
// definition fields stay authoritative from state and only existence is verified here.
func (i *impl) GetMaskingPolicy(ctx context.Context, mp *MaskingPolicy) (*MaskingPolicy, error) {
	where := []querybuilder.Where{
		querybuilder.WhereEquals("short_name", mp.Name),
		querybuilder.WhereEquals("database", mp.Database),
		querybuilder.WhereEquals("table", mp.Table),
	}
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("short_name")},
		"system.masking_policies",
	).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	found := false
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		found = true
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if !found {
		return nil, nil
	}

	out := *mp
	return &out, nil
}

func (i *impl) UpdateMaskingPolicy(ctx context.Context, mp MaskingPolicy) (*MaskingPolicy, error) {
	// CREATE OR REPLACE handles both create and update.
	return i.CreateMaskingPolicy(ctx, mp)
}

func (i *impl) DeleteMaskingPolicy(ctx context.Context, name string, database string, table string) error {
	sql, err := querybuilder.NewDropMaskingPolicy(name, database, table).IfExists(true).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	if err := i.clickhouseClient.Exec(ctx, sql); err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
