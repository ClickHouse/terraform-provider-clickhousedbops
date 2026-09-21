package dbops

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/grants"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// PartialRevoke describes a negative access-right entry in system.grants.
// It is intentionally independent of the Terraform resource so it can also be
// used by a future grantee-owned, batched privilege set.
type PartialRevoke struct {
	AccessType      string
	AccessObject    *string
	DatabaseName    *string
	TableName       *string
	ColumnName      *string
	GranteeUserName *string
	GranteeRoleName *string
	GrantOptionOnly bool
}

// AsGrant projects the target onto the neutral grants.Grant used for coverage
// checks. GrantOption is deliberately false: negative-right option coverage is
// handled separately by CoversPartialRevoke.
func (p PartialRevoke) AsGrant() grants.Grant {
	return grants.Grant{
		AccessType:   p.AccessType,
		Database:     p.DatabaseName,
		Table:        p.TableName,
		Column:       p.ColumnName,
		AccessObject: p.AccessObject,
	}
}

// CoversPartialRevoke reports whether broader makes narrower redundant.
// A full revoke also revokes the grant option; a grant-option-only revoke does
// not cover a full privilege revoke.
func CoversPartialRevoke(broader, narrower PartialRevoke) bool {
	if broader.GrantOptionOnly && !narrower.GrantOptionOnly {
		return false
	}
	return grants.Covers(broader.AsGrant(), narrower.AsGrant())
}

func partialRevokeGrantee(p PartialRevoke) (string, error) {
	switch {
	case p.GranteeUserName != nil:
		return *p.GranteeUserName, nil
	case p.GranteeRoleName != nil:
		return *p.GranteeRoleName, nil
	default:
		return "", errors.New("either GranteeUserName or GranteeRoleName must be set")
	}
}

func partialRevokeIdentifier(p PartialRevoke) string {
	identifier := p.AccessType
	if p.GranteeUserName != nil {
		return identifier + " from user " + *p.GranteeUserName
	}
	if p.GranteeRoleName != nil {
		return identifier + " from role " + *p.GranteeRoleName
	}
	return identifier
}

func (i *impl) CreatePartialRevoke(ctx context.Context, partialRevoke PartialRevoke, clusterName *string) (*PartialRevoke, error) {
	from, err := partialRevokeGrantee(partialRevoke)
	if err != nil {
		return nil, err
	}

	sql, err := querybuilder.RevokePrivilege(partialRevoke.AccessType, from).
		WithDatabase(partialRevoke.DatabaseName).
		WithTable(partialRevoke.TableName).
		WithColumn(partialRevoke.ColumnName).
		WithAccessObject(partialRevoke.AccessObject).
		WithCluster(clusterName).
		WithGrantOptionOnly(partialRevoke.GrantOptionOnly).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}
	if err := i.clickhouseClient.Exec(ctx, sql); err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	found, err := i.GetPartialRevoke(ctx, &partialRevoke, clusterName)
	if err != nil {
		return nil, err
	}
	if found != nil {
		return found, nil
	}

	covered, err := i.isPartialRevokeCovered(ctx, &partialRevoke, clusterName)
	if err != nil {
		return nil, err
	}
	if covered {
		return nil, nil
	}

	hasPositiveCoverage, err := i.hasCoveringPositiveGrant(ctx, partialRevoke, clusterName)
	if err != nil {
		return nil, err
	}
	if !hasPositiveCoverage {
		return nil, fmt.Errorf(
			"cannot create partial revoke %s: no broader positive grant covers the target",
			DescribePartialRevoke(partialRevoke),
		)
	}

	return retryWithBackoff(ctx, "partial privilege revoke", partialRevokeIdentifier(partialRevoke), func() (*PartialRevoke, error) {
		return i.GetPartialRevoke(ctx, &partialRevoke, clusterName)
	}, i.readAfterWriteTimeoutArgs()...)
}

func (i *impl) isPartialRevokeCovered(ctx context.Context, partialRevoke *PartialRevoke, clusterName *string) (bool, error) {
	existing, err := i.GetAllPartialRevokesForGrantee(ctx, partialRevoke.GranteeUserName, partialRevoke.GranteeRoleName, clusterName)
	if err != nil {
		return false, err
	}
	for idx := range existing {
		if CoversPartialRevoke(existing[idx], *partialRevoke) {
			return true, nil
		}
	}
	return false, nil
}

func partialRevokeWhere(partialRevoke *PartialRevoke) ([]querybuilder.Where, error) {
	dbName := trimWildcard(partialRevoke.DatabaseName)
	tableName := trimWildcard(partialRevoke.TableName)
	accessObject := trimWildcard(partialRevoke.AccessObject)

	where := []querybuilder.Where{
		querybuilder.WhereEquals("access_type", partialRevoke.AccessType),
		querybuilder.WhereEquals("is_partial_revoke", 1),
		querybuilder.WhereEquals("grant_option", partialRevoke.GrantOptionOnly),
		valOrNullWhere("database", dbName),
		valOrNullWhere("table", tableName),
		valOrEmptyString("access_object", accessObject),
		valOrNullWhere("column", partialRevoke.ColumnName),
	}
	switch {
	case partialRevoke.GranteeUserName != nil:
		where = append(where, querybuilder.WhereEquals("user_name", *partialRevoke.GranteeUserName))
	case partialRevoke.GranteeRoleName != nil:
		where = append(where, querybuilder.WhereEquals("role_name", *partialRevoke.GranteeRoleName))
	default:
		return nil, errors.New("either GranteeUserName or GranteeRoleName must be set")
	}
	return where, nil
}

func trimWildcard(value *string) *string {
	if value == nil || !strings.HasSuffix(*value, "*") {
		return value
	}
	return new(strings.TrimSuffix(*value, "*"))
}

func (i *impl) GetPartialRevoke(ctx context.Context, partialRevoke *PartialRevoke, clusterName *string) (*PartialRevoke, error) {
	where, err := partialRevokeWhere(partialRevoke)
	if err != nil {
		return nil, err
	}
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("access_type").ToString()},
		"system.grants",
	).WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return nil, err
	}

	found := false
	if err := i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		if _, err := data.GetString("access_type"); err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'access_type' field")
		}
		found = true
		return nil
	}); err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return partialRevoke, nil
}

func (i *impl) DeletePartialRevoke(ctx context.Context, partialRevoke PartialRevoke, clusterName *string) error {
	to, err := partialRevokeGrantee(partialRevoke)
	if err != nil {
		return err
	}

	// A refresh and a delete are not atomic. Recheck the negative row so an
	// already-removed partial revoke is an idempotent success.
	found, err := i.GetPartialRevoke(ctx, &partialRevoke, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error checking partial revoke before deletion")
	}
	if found == nil {
		return nil
	}

	// GRANT is ClickHouse's only operation for cancelling a partial revoke.
	// Refuse to execute it unless an existing positive grant covers the target:
	// otherwise a dependency/order mistake could turn deletion into privilege
	// escalation by creating a new standalone positive grant.
	covered, err := i.hasCoveringPositiveGrant(ctx, partialRevoke, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error checking covering grants before deleting partial revoke")
	}
	if !covered {
		return fmt.Errorf(
			"refusing to delete partial revoke %s: no positive grant currently covers the target; restoring it with GRANT could create a standalone privilege",
			DescribePartialRevoke(partialRevoke),
		)
	}

	// ClickHouse represents deletion of a negative access-right element by
	// granting the same target back. AccessRights normalization removes only the
	// matching partial revoke while preserving enclosing positive grants.
	sql, err := querybuilder.GrantPrivilege(partialRevoke.AccessType, to).
		WithDatabase(partialRevoke.DatabaseName).
		WithTable(partialRevoke.TableName).
		WithColumn(partialRevoke.ColumnName).
		WithAccessObject(partialRevoke.AccessObject).
		WithCluster(clusterName).
		WithGrantOption(partialRevoke.GrantOptionOnly).
		Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}
	if err := i.clickhouseClient.Exec(ctx, sql); err != nil {
		return errors.WithMessage(err, "error running query")
	}
	return nil
}

func (i *impl) hasCoveringPositiveGrant(ctx context.Context, partialRevoke PartialRevoke, clusterName *string) (bool, error) {
	positiveGrants, err := i.GetAllGrantsForGrantee(ctx, partialRevoke.GranteeUserName, partialRevoke.GranteeRoleName, clusterName)
	if err != nil {
		return false, err
	}
	target := grants.Grant{
		AccessType:   partialRevoke.AccessType,
		Database:     partialRevoke.DatabaseName,
		Table:        partialRevoke.TableName,
		Column:       partialRevoke.ColumnName,
		AccessObject: partialRevoke.AccessObject,
		GrantOption:  partialRevoke.GrantOptionOnly,
	}
	for idx := range positiveGrants {
		if grants.Covers(positiveGrants[idx].AsGrant(), target) {
			return true, nil
		}
	}
	return false, nil
}

func (i *impl) GetAllPartialRevokesForGrantee(ctx context.Context, granteeUsername *string, granteeRoleName *string, clusterName *string) ([]PartialRevoke, error) {
	where := []querybuilder.Where{querybuilder.WhereEquals("is_partial_revoke", 1)}
	switch {
	case granteeUsername != nil:
		where = append(where, querybuilder.WhereEquals("user_name", *granteeUsername))
	case granteeRoleName != nil:
		where = append(where, querybuilder.WhereEquals("role_name", *granteeRoleName))
	default:
		return nil, errors.New("either granteeUsername or granteeRoleName must be set")
	}

	sql, err := querybuilder.NewSelect([]querybuilder.Field{
		querybuilder.NewField("access_type").ToString(),
		querybuilder.NewField("database"),
		querybuilder.NewField("table"),
		querybuilder.NewField("column"),
		querybuilder.NewRawField("nullIf(access_object, '')", "access_object_nullable"),
		querybuilder.NewField("user_name"),
		querybuilder.NewField("role_name"),
		querybuilder.NewField("grant_option"),
	}, "system.grants").WithCluster(clusterName).Where(where...).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	result := make([]PartialRevoke, 0)
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
		userName, err := data.GetNullableString("user_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'user_name' field")
		}
		roleName, err := data.GetNullableString("role_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'role_name' field")
		}
		grantOptionOnly, err := data.GetBool("grant_option")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'grant_option' field")
		}
		result = append(result, PartialRevoke{
			AccessType:      accessType,
			AccessObject:    accessObject,
			DatabaseName:    database,
			TableName:       table,
			ColumnName:      column,
			GranteeUserName: userName,
			GranteeRoleName: roleName,
			GrantOptionOnly: grantOptionOnly,
		})
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}
	return result, nil
}

func DescribePartialRevoke(p PartialRevoke) string {
	target := "*.*"
	switch {
	case p.AccessObject != nil:
		target = *p.AccessObject
	case p.DatabaseName != nil && p.TableName != nil && p.ColumnName != nil:
		target = fmt.Sprintf("%s.%s(%s)", *p.DatabaseName, *p.TableName, *p.ColumnName)
	case p.DatabaseName != nil && p.TableName != nil:
		target = fmt.Sprintf("%s.%s", *p.DatabaseName, *p.TableName)
	case p.DatabaseName != nil:
		target = *p.DatabaseName + ".*"
	}
	return fmt.Sprintf("%s on %s", p.AccessType, target)
}
