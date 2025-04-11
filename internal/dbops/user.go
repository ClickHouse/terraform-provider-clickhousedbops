package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type User struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	PasswordSha256Hash string `json:"-"`
}

func (i *impl) CreateUser(ctx context.Context, user User) (*User, error) {
	sql, err := querybuilder.
		NewCreateUser(user.Name).
		Identified(querybuilder.IdentificationSHA256Hash, user.PasswordSha256Hash).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	// Get ID of newly created user
	var id string
	{
		sql, err := querybuilder.NewSelect(
			[]querybuilder.Field{querybuilder.NewField("id")},
			"system.users",
		).Where(querybuilder.SimpleWhere("name", user.Name)).Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}

		err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
			id, err = data.GetString("id")
			if err != nil {
				return errors.WithMessage(err, "error scanning query result, missing 'id' field")
			}

			return nil
		})
		if err != nil {
			return nil, errors.WithMessage(err, "error running query")
		}
	}

	createdUser, err := i.GetUser(ctx, id)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting user")
	}

	return createdUser, nil
}

func (i *impl) GetUser(ctx context.Context, id string) (*User, error) { // nolint:dupl
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name")},
		"system.users",
	).Where(querybuilder.SimpleWhere("id", id)).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var user *User

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		n, err := data.GetString("name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'name' field")
		}
		user = &User{
			ID:   id,
			Name: n,
		}
		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if user == nil {
		// User not found
		return nil, nil
	}

	return user, nil
}

func (i *impl) DeleteUser(ctx context.Context, id string) error {
	user, err := i.GetUser(ctx, id)
	if err != nil {
		return errors.WithMessage(err, "error getting user")
	}

	if user == nil {
		// This is the desired state.
		return nil
	}

	sql, err := querybuilder.NewDropUser(user.Name).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}
