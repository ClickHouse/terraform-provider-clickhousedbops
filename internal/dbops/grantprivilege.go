package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// Source family, used in GetGrantPrivilege logic below to cover
// transition to new grant model for external datasources with separation on READ/WRITE grants.
var sourcesFamily = map[string]bool{
	"AZURE":    true,
	"FILE":     true,
	"HDFS":     true,
	"HIVE":     true,
	"JDBC":     true,
	"KAFKA":    true,
	"MONGO":    true,
	"MYSQL":    true,
	"NATS":     true,
	"ODBC":     true,
	"POSTGRES": true,
	"RABBITMQ": true,
	"REDIS":    true,
	"REMOTE":   true,
	"S3":       true,
	"SQLITE":   true,
	"URL":      true,
}

type GrantPrivilege struct {
	AccessType      string  `json:"access_type"`
	AccessObject    string  `json:"access_object"`
	DatabaseName    *string `json:"database"`
	TableName       *string `json:"table"`
	ColumnName      *string `json:"column"`
	GranteeUserName *string `json:"user_name"`
	GranteeRoleName *string `json:"role_name"`
	GrantOption     bool    `json:"grant_option"`
}

// Defines the signature for a function that checks if privileges are granted.
type MatcherFunc func(ctx context.Context, priv *GrantPrivilege, clusterName *string, i *impl) (bool, error)

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

	identifier := grantPrivilege.AccessType
	if grantPrivilege.GranteeUserName != nil {
		identifier += " to user " + *grantPrivilege.GranteeUserName
	} else if grantPrivilege.GranteeRoleName != nil {
		identifier += " to role " + *grantPrivilege.GranteeRoleName
	}

	return retryWithBackoff(ctx, "grant privilege", identifier, func() (*GrantPrivilege, error) {
		return i.GetGrantPrivilege(ctx, &grantPrivilege, clusterName)
	})
}

// Matcher function to handle classic grants: https://clickhouse.com/docs/sql-reference/statements/grant#granting-privilege-syntax
func ClassicGrantMatcher(ctx context.Context, priv *GrantPrivilege, clusterName *string, i *impl) (bool, error) {
	where := []querybuilder.Where{
		querybuilder.WhereEquals("access_type", priv.AccessType),
		valOrNullWhere("database", priv.DatabaseName),
		valOrNullWhere("table", priv.TableName),
		valOrNullWhere("column", priv.ColumnName),
	}
	if priv.GranteeUserName != nil {
		where = append(where, querybuilder.WhereEquals("user_name", *priv.GranteeUserName))
	} else if priv.GranteeRoleName != nil {
		where = append(where, querybuilder.WhereEquals("role_name", *priv.GranteeRoleName))
	} else {
		return false, errors.New("either GranteeUserName or GranteeRoleName must be set")
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
		return false, err
	}

	found := false
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		_, err = data.GetString("access_type")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'access_type' field")
		}
		_, err = data.GetNullableString("database")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'database' field")
		}
		_, err = data.GetNullableString("table")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'table' field")
		}
		_, err = data.GetNullableString("column")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'column' field")
		}
		_, err = data.GetNullableString("user_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'user_name' field")
		}
		_, err = data.GetNullableString("role_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'role_name' field")
		}
		_, err = data.GetBool("grant_option")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'grant_option' field")
		}
		found = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return found, nil
}

// Matcher function to handle sources grants, applied via `clickhousedbops_grant_privilege` resource with READ/WRITE access:
// https://clickhouse.com/docs/sql-reference/statements/grant#sources
// TODO: grants for sources should be refactored to use separate resource.
func SourcesReadWriteGrantMatcher(ctx context.Context, priv *GrantPrivilege, clusterName *string, i *impl) (bool, error) {
	if priv.AccessObject == "" {
		return false, errors.New("incorrect query: access_object field required for sources matcher")
	}
	where := []querybuilder.Where{
		querybuilder.WhereEquals("access_object", priv.AccessObject),
		querybuilder.WhereIn("access_type", []string{"READ", "WRITE"}),
		valOrNullWhere("database", priv.DatabaseName),
		valOrNullWhere("table", priv.TableName),
		valOrNullWhere("column", priv.ColumnName),
	}
	if priv.GranteeUserName != nil {
		where = append(where, querybuilder.WhereEquals("user_name", *priv.GranteeUserName))
	} else if priv.GranteeRoleName != nil {
		where = append(where, querybuilder.WhereEquals("role_name", *priv.GranteeRoleName))
	} else {
		return false, errors.New("incorrect query: either user_name or role_name must be set")
	}

	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("access_type").ToString(),
			querybuilder.NewField("access_object"),
			querybuilder.NewField("user_name"),
			querybuilder.NewField("role_name"),
			querybuilder.NewField("grant_option"),
		},
		"system.grants",
	).WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return false, err
	}
	// We expect 2 rows for both READ and WRITE grants.
	rowsCount := 0
	err = i.clickhouseClient.Select(ctx, sql, func(_ clickhouseclient.Row) error {
		rowsCount++
		return nil
	})
	if err != nil {
		return false, err
	}

	return rowsCount == 2, nil
}

// Helper function: Null or value clause
func valOrNullWhere(field string, value *string) querybuilder.Where {
	if value != nil {
		return querybuilder.WhereEquals(field, *value)
	}
	return querybuilder.IsNull(field)
}

func (i *impl) GetGrantPrivilege(ctx context.Context, grantPrivilege *GrantPrivilege, clusterName *string) (*GrantPrivilege, error) {
	var matcher MatcherFunc
	capabilityFlags, err := i.GetCapabilityFlags(ctx)
	if err != nil {
		return nil, err
	}
	// Use sources matcher if capability and accessType is a source, otherwise classic one
	if capabilityFlags.SourcesGrantReadWriteSeparation && sourcesFamily[grantPrivilege.AccessType] {
		grantPrivilege.AccessObject = grantPrivilege.AccessType
		matcher = SourcesReadWriteGrantMatcher
	} else {
		matcher = ClassicGrantMatcher
	}

	ok, err := matcher(ctx, grantPrivilege, clusterName, i)
	if err != nil {
		return nil, err
	}
	if !ok {
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
