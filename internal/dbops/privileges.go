package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// readPrivileges reads the group hierarchy and scopes; aliases only matter for validating configuration, which uses the bundled catalog.
func (i *impl) readPrivileges(ctx context.Context) (grants.Catalog, error) {
	sql, err := querybuilder.NewSelect([]querybuilder.Field{
		querybuilder.NewField("privilege").ToString(),
		querybuilder.NewField("level").ToString(),
		querybuilder.NewField("parent_group").ToString(),
	}, "system.privileges").Build()
	if err != nil {
		return grants.Catalog{}, errors.WithMessage(err, "error building query")
	}

	privileges := make([]grants.Privilege, 0)
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		name, err := data.GetString("privilege")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'privilege' field")
		}
		level, err := data.GetNullableString("level")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'level' field")
		}
		parentGroup, err := data.GetNullableString("parent_group")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'parent_group' field")
		}

		privileges = append(privileges, grants.Privilege{Name: name, Scope: level, ParentGroup: parentGroup})
		return nil
	})
	if err != nil {
		return grants.Catalog{}, errors.WithMessage(err, "error running query")
	}

	return grants.NewCatalog(privileges), nil
}
