package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type GrantRole struct {
	RoleName        string  `json:"granted_role_name"`
	GranteeUserName *string `json:"user_name"`
	GranteeRoleName *string `json:"role_name"`
	AdminOption     bool    `json:"with_admin_option"`
}

func (i *impl) GrantRole(ctx context.Context, grantRole GrantRole, clusterName *string) (*GrantRole, error) {
	var to string
	{
		if grantRole.GranteeUserName != nil {
			to = *grantRole.GranteeUserName
		} else if grantRole.GranteeRoleName != nil {
			to = *grantRole.GranteeRoleName
		} else {
			return nil, errors.New("either GranteeUserName or GranteeRoleName must be set")
		}
	}

	sql, err := querybuilder.GrantRole(grantRole.RoleName, to).WithCluster(clusterName).WithAdminOption(grantRole.AdminOption).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.GetGrantRole(ctx, grantRole.RoleName, grantRole.GranteeUserName, grantRole.GranteeRoleName, clusterName)
}

func (i *impl) GetGrantRole(ctx context.Context, grantedRoleName string, granteeUserName *string, granteeRoleName *string, clusterName *string) (*GrantRole, error) {
	var granteeWhere querybuilder.Where
	{
		if granteeUserName != nil {
			granteeWhere = querybuilder.WhereEquals("user_name", *granteeUserName)
		} else if granteeRoleName != nil {
			granteeWhere = querybuilder.WhereEquals("role_name", *granteeRoleName)
		} else {
			return nil, errors.New("either GranteeUserName or GranteeRoleName must be set")
		}
	}

	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("granted_role_name"),
			querybuilder.NewField("user_name"),
			querybuilder.NewField("role_name"),
			querybuilder.NewField("with_admin_option"),
		},
		"system.role_grants").
		WithCluster(clusterName).
		Where(querybuilder.WhereEquals("granted_role_name", grantedRoleName), granteeWhere).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var grantRole *GrantRole

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		roleName, err := data.GetString("granted_role_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'name' field")
		}
		granteeUserName, err := data.GetNullableString("user_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'user_name' field")
		}
		granteeRoleName, err := data.GetNullableString("role_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'role_name' field")
		}
		adminOption, err := data.GetBool("with_admin_option")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'with_admin_option' field")
		}
		grantRole = &GrantRole{
			RoleName:        roleName,
			GranteeUserName: granteeUserName,
			GranteeRoleName: granteeRoleName,
			AdminOption:     adminOption,
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if grantRole == nil {
		// Grant not found
		return nil, nil
	}

	return grantRole, nil
}

func (i *impl) RevokeGrantRole(ctx context.Context, grantedRoleName string, granteeUserName *string, granteeRoleName *string, clusterName *string) error {
	var grantee string
	{
		if granteeUserName != nil {
			grantee = *granteeUserName
		} else if granteeRoleName != nil {
			grantee = *granteeRoleName
		} else {
			return errors.New("either GranteeUserName or GranteeRoleName must be set")
		}
	}
	sql, err := querybuilder.RevokeRole(grantedRoleName, grantee).WithCluster(clusterName).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
