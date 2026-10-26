package dbops

import (
	"context"
	"fmt"
	"regexp"
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
	AccessType      string  `json:"access_type"`
	AccessObject    *string `json:"access_object"`
	SourceFilter    *string `json:"-"`
	DatabaseName    *string `json:"database"`
	TableName       *string `json:"table"`
	ColumnName      *string `json:"column"`
	GranteeUserName *string `json:"user_name"`
	GranteeRoleName *string `json:"role_name"`
	GrantOption     bool    `json:"grant_option"`
	// ExpandedAccessTypes includes AccessType and all its descendants.
	// ClickHouse may expand a parent privilege (e.g. CREATE, ACCESS MANAGEMENT)
	// into its children in system.grants instead of storing a single parent row.
	// Setting this field lets GetGrantPrivilege find the grant in either case.
	ExpandedAccessTypes []string `json:"-"`
	// CurrentGrants emits `GRANT CURRENT GRANTS(... ON ...)`, copying the grantor's own
	// privileges. Needed on ClickHouse Cloud for broad grants the default admin holds but
	// cannot transfer directly (see #190).
	CurrentGrants bool `json:"-"`
}

// AsGrant projects the grant onto the neutral grants.Grant used for coverage checks.
func (g GrantPrivilege) AsGrant() grants.Grant {
	return grants.Grant{
		AccessType:   g.AccessType,
		Database:     g.DatabaseName,
		Table:        g.TableName,
		Column:       g.ColumnName,
		AccessObject: g.AccessObject,
		GrantOption:  g.GrantOption,
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

	sql, err := querybuilder.GrantPrivilege(grantPrivilege.AccessType, to).
		WithDatabase(grantPrivilege.DatabaseName).
		WithTable(grantPrivilege.TableName).
		WithColumn(grantPrivilege.ColumnName).
		WithAccessObject(grantPrivilege.AccessObject).
		WithSourceFilter(grantPrivilege.SourceFilter).
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

	// Wait for the grant's rows, or for a broader grant that absorbed it, in which case ClickHouse writes none.
	return retryWithBackoff(ctx, "grant privilege", identifier, func() (*GrantPrivilege, error) {
		return i.GetGrantPrivilege(ctx, &grantPrivilege, clusterName, false)
	}, i.readAfterWriteTimeoutArgs()...)
}

func (i *impl) GetGrantPrivilege(ctx context.Context, grantPrivilege *GrantPrivilege, clusterName *string, allowCached bool) (*GrantPrivilege, error) {
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
	accessObjects := []string{name(grantPrivilege.AccessObject)}
	if grantPrivilege.SourceFilter != nil {
		accessObjects = sourceFilterTargets(*grantPrivilege.AccessObject, *grantPrivilege.SourceFilter)
	}

	isReadWrite := false
	var accessTypes []string
	switch {
	case capabilityFlags.SourcesGrantReadWriteSeparation && sourcesFamily[grantPrivilege.AccessType]:
		accessTypes = []string{"READ", "WRITE"}
		accessObjects = []string{grantPrivilege.AccessType}
		isReadWrite = true
	case len(grantPrivilege.ExpandedAccessTypes) > 0:
		accessTypes = grantPrivilege.ExpandedAccessTypes
	default:
		accessTypes = []string{grantPrivilege.AccessType}
	}

	existing, err := i.grantsForGrantee(ctx, grantPrivilege.GranteeUserName, grantPrivilege.GranteeRoleName, clusterName, allowCached)
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, row := range existing {
		if slices.Contains(accessTypes, row.AccessType) &&
			name(row.DatabaseName) == database &&
			name(row.TableName) == table &&
			name(row.ColumnName) == column &&
			slices.Contains(accessObjects, name(row.AccessObject)) {
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
		WithSourceFilter(grantPrivilege.SourceFilter).
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

		// Without is_wildcard any stored name may be a pattern, except a source.
		isWildcard := grants.Parsed().Scopes[accessType] != "SOURCE"
		if capabilityFlags.GrantsWildcardColumn {
			isWildcard, err = data.GetBool("is_wildcard")
			if err != nil {
				return errors.WithMessage(err, "error scanning query result, missing 'is_wildcard' field")
			}
		}
		// The pattern is the deepest set dimension; restore its trailing '*' as written in config.
		switch {
		case !isWildcard:
		case accessObject != nil:
			accessObject = new(*accessObject + "*")
		case table != nil:
			table = new(*table + "*")
		case database != nil:
			database = new(*database + "*")
		}

		ret = append(ret, GrantPrivilege{
			AccessType:      accessType,
			AccessObject:    accessObject,
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

// grantsForGrantee with allowCached reads each grantee once until the next Exec clears the cache.
func (i *impl) grantsForGrantee(ctx context.Context, granteeUserName, granteeRoleName, clusterName *string, allowCached bool) ([]GrantPrivilege, error) {
	if !allowCached {
		return i.GetAllGrantsForGrantee(ctx, granteeUserName, granteeRoleName, clusterName)
	}

	key := grantsCacheKey(granteeUserName, granteeRoleName, clusterName)
	result, err, _ := i.grantsFills.Do(key, func() (any, error) {
		if rows, ok := i.grantsCache.Load(key); ok {
			return rows, nil
		}
		rows, err := i.GetAllGrantsForGrantee(ctx, granteeUserName, granteeRoleName, clusterName)
		if err != nil {
			return nil, err
		}
		i.grantsCache.Store(key, rows)
		return rows, nil
	})
	if err != nil {
		return nil, err
	}
	return result.([]GrantPrivilege), nil
}

func grantsCacheKey(granteeUserName, granteeRoleName, clusterName *string) string {
	var result string
	if clusterName != nil {
		result = *clusterName + ":"
	}

	switch {
	case granteeUserName != nil:
		result += "user:" + *granteeUserName
	case granteeRoleName != nil:
		result += "role:" + *granteeRoleName
	}

	return result
}

var plainIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// sourceFilterTargets renders a filtered source as system.grants shows it; identifier-like filters may appear unquoted.
func sourceFilterTargets(source, filter string) []string {
	targets := []string{fmt.Sprintf("%s(%s)", source, querybuilder.QuoteChar(filter, '`'))}
	if plainIdentifier.MatchString(filter) {
		targets = append(targets, fmt.Sprintf("%s(%s)", source, filter))
	}
	return targets
}
