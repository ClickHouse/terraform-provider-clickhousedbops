package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Role struct {
	ID   string `json:"id" ch:"id"`
	Name string `json:"name" ch:"name"`
}

func (i *impl) CreateRole(ctx context.Context, role Role) (*Role, error) {
	sql, err := querybuilder.NewCreateRole(role.Name).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	// Get ID of newly created role.
	var id string
	{
		sql, err := querybuilder.NewSelect(
			[]querybuilder.Field{querybuilder.NewField("id")},
			"system.roles",
		).With(querybuilder.Where("name", role.Name)).Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}

		err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
			id, err = data.Get("id")
			if err != nil {
				return errors.WithMessage(err, "error scanning query result, missing 'id' field")
			}

			return nil
		})
		if err != nil {
			return nil, errors.WithMessage(err, "error running query")
		}
	}

	return i.GetRole(ctx, id)
}

func (i *impl) GetRole(ctx context.Context, id string) (*Role, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name")},
		"system.roles",
	).With(querybuilder.Where("id", id)).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var role *Role

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		n, err := data.Get("name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'name' field")
		}
		role = &Role{
			ID:   id,
			Name: n,
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if role == nil {
		// Role not found
		return nil, nil
	}

	return role, nil
}

func (i *impl) DeleteRole(ctx context.Context, id string) error {
	role, err := i.GetRole(ctx, id)
	if err != nil {
		return errors.WithMessage(err, "error getting role")
	}

	if role == nil {
		// That's what we want.
		return nil
	}

	sql, err := querybuilder.NewDropRole(role.Name).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
