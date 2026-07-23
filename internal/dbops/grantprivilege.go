package dbops

import (
	"context"
	"slices"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// Legacy source privileges, which ClickHouse 25.7+ stores as READ and WRITE on the source.
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
	AccessType         string  `json:"access_type"`
	AccessObject       *string `json:"access_object"`
	AccessObjectFilter *string `json:"-"`
	DatabaseName       *string `json:"database"`
	TableName          *string `json:"table"`
	ColumnName         *string `json:"column"`
	GranteeUserName    *string `json:"user_name"`
	GranteeRoleName    *string `json:"role_name"`
	GrantOption        bool    `json:"grant_option"`
	// ExpandedAccessTypes includes AccessType and all its descendants.
	// ClickHouse may expand a parent privilege (e.g. CREATE, ACCESS MANAGEMENT)
	// into its children in system.grants instead of storing a single parent row.
	// Setting this field lets GetGrantPrivilege find the grant in either case.
	ExpandedAccessTypes []string `json:"-"`
	// CurrentGrants emits `GRANT CURRENT GRANTS(... ON ...)`, copying the grantor's own
	// privileges. Needed on ClickHouse Cloud for broad grants the default admin holds but
	// cannot transfer directly (see #190).
	CurrentGrants bool `json:"-"`
	// ParameterizedTarget selects ClickHouse's global-with-parameter ON syntax.
	// Unlike database/table privileges, an unrestricted parameter target is ON *.
	ParameterizedTarget bool `json:"-"`
}

// AsGrant projects the grant onto the neutral grants.Grant used for coverage checks.
func (g GrantPrivilege) AsGrant() grants.Grant {
	return grants.Grant{
		AccessType:         g.AccessType,
		Database:           g.DatabaseName,
		Table:              g.TableName,
		Column:             g.ColumnName,
		AccessObject:       g.AccessObject,
		AccessObjectFilter: g.AccessObjectFilter,
		GrantOption:        g.GrantOption,
	}
}

// OverlapError reports existing grants that include the grant, so it has no row of its own.
type OverlapError struct {
	Covering []GrantPrivilege
}

func (e *OverlapError) Error() string {
	return "the privilege is already included in an existing grant"
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

	hadUnfilteredSourceGrant := false
	if grantPrivilege.AccessObjectFilter != nil {
		unfilteredGrant := grantPrivilege
		unfilteredGrant.AccessObjectFilter = nil
		existing, err := i.GetGrantPrivilege(ctx, &unfilteredGrant, clusterName)
		if err != nil {
			return nil, errors.WithMessage(err, "error checking for an existing unfiltered source grant")
		}
		hadUnfilteredSourceGrant = existing != nil
	}

	sql, err := querybuilder.GrantPrivilege(grantPrivilege.AccessType, to).
		WithDatabase(grantPrivilege.DatabaseName).
		WithTable(grantPrivilege.TableName).
		WithColumn(grantPrivilege.ColumnName).
		WithAccessObject(grantPrivilege.AccessObject).
		WithAccessObjectFilter(grantPrivilege.AccessObjectFilter).
		WithParameterizedTarget(grantPrivilege.ParameterizedTarget).
		WithGrantOption(grantPrivilege.GrantOption).
		WithCluster(clusterName).
		WithCurrentGrants(grantPrivilege.CurrentGrants).
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

	// With access_control_improvements.enable_read_write_grants disabled,
	// ClickHouse accepts filtered READ/WRITE syntax but records an unfiltered
	// source grant. Never adopt that broader grant as the requested resource.
	if grantPrivilege.AccessObjectFilter != nil {
		if found, err := i.GetGrantPrivilege(ctx, &grantPrivilege, clusterName); err == nil && found != nil {
			return found, nil
		}
		unfilteredGrant := grantPrivilege
		unfilteredGrant.AccessObjectFilter = nil
		unfiltered, err := i.GetGrantPrivilege(ctx, &unfilteredGrant, clusterName)
		if err != nil {
			return nil, errors.WithMessage(err, "error checking whether ClickHouse preserved the source filter")
		}
		if unfiltered != nil {
			detail := "ClickHouse recorded the filtered source grant as an unfiltered grant. Enable access_control_improvements.enable_read_write_grants on the server before managing source filters."
			if !hadUnfilteredSourceGrant {
				if err := i.RevokeGrantPrivilege(ctx, unfilteredGrant, clusterName); err != nil {
					return nil, errors.Wrapf(err, "%s The provider also failed to roll back the broader grant", detail)
				}
				detail += " The provider rolled back the broader grant."
			}
			return nil, errors.New(detail)
		}
	}

	// Wait for the grant's rows, or for a broader grant that absorbed it, in which case ClickHouse writes none.
	return retryWithBackoff(ctx, "grant privilege", identifier, func() (*GrantPrivilege, error) {
		return i.GetGrantPrivilege(ctx, &grantPrivilege, clusterName)
	}, i.readAfterWriteTimeoutArgs()...)
}

func (i *impl) GetGrantPrivilege(ctx context.Context, grantPrivilege *GrantPrivilege, clusterName *string) (*GrantPrivilege, error) {
	capabilityFlags, err := i.GetCapabilityFlags(ctx)
	if err != nil {
		return nil, err
	}

	// Since 26.7 stored names keep the '*' of prefix grants; before, any stored name may be a prefix.
	name := func(s *string) string {
		switch {
		case s == nil:
			return ""
		case capabilityFlags.GrantsWildcardColumn:
			return *s
		default:
			return strings.TrimSuffix(*s, "*")
		}
	}

	database := name(grantPrivilege.DatabaseName)
	table := name(grantPrivilege.TableName)
	column := name(grantPrivilege.ColumnName)
	accessObject := name(grantPrivilege.AccessObject)

	isReadWrite := false
	var accessTypes []string
	switch {
	case capabilityFlags.SourcesGrantReadWriteSeparation && sourcesFamily[grantPrivilege.AccessType]:
		accessTypes = []string{"READ", "WRITE"}
		accessObject = grantPrivilege.AccessType
		isReadWrite = true
	case len(grantPrivilege.ExpandedAccessTypes) > 0:
		accessTypes = grantPrivilege.ExpandedAccessTypes
	default:
		accessTypes = []string{grantPrivilege.AccessType}
	}

	existing, err := i.GetAllGrantsForGrantee(ctx, grantPrivilege.GranteeUserName, grantPrivilege.GranteeRoleName, clusterName)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, row := range existing {
		if slices.Contains(accessTypes, row.AccessType) &&
			name(row.DatabaseName) == database &&
			name(row.TableName) == table &&
			name(row.ColumnName) == column &&
			name(row.AccessObject) == accessObject &&
			name(row.AccessObjectFilter) == name(grantPrivilege.AccessObjectFilter) {
			seen[row.AccessType] = true
		}
	}

	if (isReadWrite && len(seen) == len(accessTypes)) || (!isReadWrite && len(seen) > 0) {
		return grantPrivilege, nil
	}

	var covering []GrantPrivilege
	for _, e := range existing {
		if grants.Covers(e.AsGrant(), grantPrivilege.AsGrant()) {
			covering = append(covering, e)
		}
	}
	if len(covering) > 0 {
		return nil, &OverlapError{Covering: covering}
	}
	return nil, nil
}

func (i *impl) RevokeGrantPrivilege(ctx context.Context, grantPrivilege GrantPrivilege, clusterName *string) error {
	var from string
	{
		switch {
		case grantPrivilege.GranteeUserName != nil:
			from = *grantPrivilege.GranteeUserName
		case grantPrivilege.GranteeRoleName != nil:
			from = *grantPrivilege.GranteeRoleName
		default:
			return errors.New("either GranteeUserName or GranteeRoleName must be set")
		}
	}

	sql, err := querybuilder.RevokePrivilege(grantPrivilege.AccessType, from).
		WithDatabase(grantPrivilege.DatabaseName).
		WithTable(grantPrivilege.TableName).
		WithColumn(grantPrivilege.ColumnName).
		WithAccessObject(grantPrivilege.AccessObject).
		WithAccessObjectFilter(grantPrivilege.AccessObjectFilter).
		WithParameterizedTarget(grantPrivilege.ParameterizedTarget).
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
	where := []querybuilder.Where{querybuilder.WhereEquals("is_partial_revoke", 0)}
	{
		if granteeUsername != nil {
			where = append(where, querybuilder.WhereEquals("user_name", *granteeUsername))
		} else if granteeRoleName != nil {
			where = append(where, querybuilder.WhereEquals("role_name", *granteeRoleName))
		} else {
			return nil, errors.New("either granteeUsername or GranteeRoleName must be set")
		}
	}

	capabilityFlags, err := i.GetCapabilityFlags(ctx)
	if err != nil {
		return nil, err
	}

	fields := []querybuilder.Field{
		querybuilder.NewField("access_type").ToString(),
		querybuilder.NewField("database"),
		querybuilder.NewField("table"),
		querybuilder.NewField("column"),
		// Alias must differ from the column name or ClickHouse substitutes it into WHERE.
		querybuilder.NewRawField("nullIf(access_object, '')", "access_object_nullable"),
		querybuilder.NewField("user_name"),
		querybuilder.NewField("role_name"),
		querybuilder.NewField("grant_option"),
	}
	if capabilityFlags.GrantsWildcardColumn {
		fields = append(fields, querybuilder.NewField("is_wildcard"))
	}

	sql, err := querybuilder.NewSelect(fields, "system.grants").
		WithCluster(clusterName).
		Where(querybuilder.AndWhere(where...)).
		Build()
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
		accessObject, err := data.GetNullableString("access_object_nullable")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'access_object_nullable' field")
		}
		accessObject, accessObjectFilter := splitSystemAccessObject(accessType, accessObject)
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

		isWildcard := true
		if capabilityFlags.GrantsWildcardColumn {
			isWildcard, err = data.GetBool("is_wildcard")
			if err != nil {
				return errors.WithMessage(err, "error scanning query result, missing 'is_wildcard' field")
			}
		}
		// The pattern is the deepest set dimension; restore its trailing '*' as written in config. A filtered source is never a pattern.
		switch {
		case !isWildcard || accessObjectFilter != nil:
		case accessObject != nil:
			accessObject = new(*accessObject + "*")
		case table != nil:
			table = new(*table + "*")
		case database != nil:
			database = new(*database + "*")
		}

		ret = append(ret, GrantPrivilege{
			AccessType:         accessType,
			AccessObject:       accessObject,
			AccessObjectFilter: accessObjectFilter,
			DatabaseName:       database,
			TableName:          table,
			ColumnName:         column,
			GranteeUserName:    granteeUserName,
			GranteeRoleName:    granteeRoleName,
			GrantOption:        grantOption,
		})

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return ret, nil
}
