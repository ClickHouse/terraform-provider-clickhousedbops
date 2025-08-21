package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type GrantPrivilege struct {
	AccessType      string  `json:"access_type"`
	DatabaseName    *string `json:"database"`
	TableName       *string `json:"table"`
	ColumnName      *string `json:"column"`
	GranteeUserName *string `json:"user_name"`
	GranteeRoleName *string `json:"role_name"`
	GrantOption     bool    `json:"grant_option"`
}

func (i *impl) GrantPrivilege(ctx context.Context, grantPrivilege GrantPrivilege, clusterName *string) (*GrantPrivilege, error) {
	var to string
	{
		if grantPrivilege.GranteeUserName != nil {
			to = *grantPrivilege.GranteeUserName
		} else if grantPrivilege.GranteeRoleName != nil {
			to = *grantPrivilege.GranteeRoleName
		} else {
			return nil, errors.New("either GranteeUserName or GranteeRoleName must be set")
		}
	}

	sql, err := querybuilder.GrantPrivilege(grantPrivilege.AccessType, to).
		WithDatabase(grantPrivilege.DatabaseName).
		WithTable(grantPrivilege.TableName).
		WithColumn(grantPrivilege.ColumnName).
		WithGrantOption(grantPrivilege.GrantOption).
		WithCluster(clusterName).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.GetGrantPrivilege(ctx, grantPrivilege.AccessType, grantPrivilege.DatabaseName, grantPrivilege.TableName, grantPrivilege.ColumnName, grantPrivilege.GranteeUserName, grantPrivilege.GranteeRoleName, clusterName)
}

func (i *impl) GetGrantPrivilege(ctx context.Context, accessType string, database *string, table *string, column *string, granteeUserName *string, granteeRoleName *string, clusterName *string) (*GrantPrivilege, error) {
	where := make([]querybuilder.Where, 0)

	{
		where = append(where, querybuilder.WhereEquals("access_type", accessType))
		if database != nil {
			where = append(where, querybuilder.WhereEquals("database", *database))
		} else {
			where = append(where, querybuilder.IsNull("database"))
		}

		if table != nil {
			where = append(where, querybuilder.WhereEquals("table", *table))
		} else {
			where = append(where, querybuilder.IsNull("table"))
		}

		if column != nil {
			where = append(where, querybuilder.WhereEquals("column", *column))
		} else {
			where = append(where, querybuilder.IsNull("column"))
		}

		if granteeUserName != nil {
			where = append(where, querybuilder.WhereEquals("user_name", *granteeUserName))
		} else if granteeRoleName != nil {
			where = append(where, querybuilder.WhereEquals("role_name", *granteeRoleName))
		} else {
			return nil, errors.New("either GranteeUserName or GranteeRoleName must be set")
		}
	}

	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("access_type").ToString(),
			querybuilder.NewField("database"),
			querybuilder.NewField("table"),
			querybuilder.NewField("column"),
			querybuilder.NewField("user_name"),
			querybuilder.NewField("role_name"),
			querybuilder.NewField("grant_option"),
		},
		"system.grants",
	).WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var grantPrivilege *GrantPrivilege

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		accessType, err := data.GetString("access_type")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'access_type' field")
		}
		database, err := data.GetNullableString("database")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'database' field")
		}
		table, err := data.GetNullableString("table")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'table' field")
		}
		column, err := data.GetNullableString("column")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'column' field")
		}
		granteeUserName, err := data.GetNullableString("user_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'user_name' field")
		}
		granteeRoleName, err := data.GetNullableString("role_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'role_name' field")
		}
		grantOption, err := data.GetBool("grant_option")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'grant_option' field")
		}
		grantPrivilege = &GrantPrivilege{
			AccessType:      accessType,
			DatabaseName:    database,
			TableName:       table,
			ColumnName:      column,
			GranteeUserName: granteeUserName,
			GranteeRoleName: granteeRoleName,
			GrantOption:     grantOption,
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if grantPrivilege == nil {
		// Grant not found
		return nil, nil
	}

	return grantPrivilege, nil
}

func (i *impl) RevokeGrantPrivilege(ctx context.Context, accessType string, database *string, table *string, column *string, granteeUserName *string, granteeRoleName *string, clusterName *string) error {
	var from string
	{
		if granteeUserName != nil {
			from = *granteeUserName
		} else if granteeRoleName != nil {
			from = *granteeRoleName
		} else {
			return errors.New("either GranteeUserName or GranteeRoleName must be set")
		}
	}

	sql, err := querybuilder.RevokePrivilege(accessType, from).
		WithDatabase(database).
		WithTable(table).
		WithColumn(column).
		WithCluster(clusterName).
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

func (i *impl) GetAllGrantsForGrantee(ctx context.Context, granteeUsername *string, granteeRoleName *string, clusterName *string) ([]GrantPrivilege, error) {
	// Get all grants for the same grantee.
	var to querybuilder.Where
	{
		if granteeUsername != nil {
			to = querybuilder.WhereEquals("user_name", *granteeUsername)
		} else if granteeRoleName != nil {
			to = querybuilder.WhereEquals("role_name", *granteeRoleName)
		} else {
			return nil, errors.New("either granteeUsername or GranteeRoleName must be set")
		}
	}

	sql, err := querybuilder.NewSelect([]querybuilder.Field{
		querybuilder.NewField("access_type").ToString(),
		querybuilder.NewField("database"),
		querybuilder.NewField("table"),
		querybuilder.NewField("column"),
		querybuilder.NewField("user_name"),
		querybuilder.NewField("role_name"),
		querybuilder.NewField("grant_option"),
	}, "system.grants").WithCluster(clusterName).Where(to).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	ret := make([]GrantPrivilege, 0)

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		accessType, err := data.GetString("access_type")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'access_type' field")
		}
		database, err := data.GetNullableString("database")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'database' field")
		}
		table, err := data.GetNullableString("table")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'table' field")
		}
		column, err := data.GetNullableString("column")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'column' field")
		}
		granteeUserName, err := data.GetNullableString("user_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'user_name' field")
		}
		granteeRoleName, err := data.GetNullableString("role_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'role_name' field")
		}
		grantOption, err := data.GetBool("grant_option")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'grant_option' field")
		}

		ret = append(ret, GrantPrivilege{
			AccessType:      accessType,
			DatabaseName:    database,
			TableName:       table,
			ColumnName:      column,
			GranteeUserName: granteeUserName,
			GranteeRoleName: granteeRoleName,
			GrantOption:     grantOption,
		})

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return ret, nil
}
