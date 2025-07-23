package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type User struct {
	ID                 string  `json:"id"`
	Name               string  `json:"name"`
	PasswordSha256Hash string  `json:"-"`
	SettingsProfile    *string `json:"-"`
}

func (i *impl) CreateUser(ctx context.Context, user User, clusterName *string) (*User, error) {
	sql, err := querybuilder.
		NewCreateUser(user.Name).
		Identified(querybuilder.IdentificationSHA256Hash, user.PasswordSha256Hash).
		WithCluster(clusterName).
		WithSettingsProfile(user.SettingsProfile).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.FindUserByName(ctx, user.Name, clusterName)
}

func (i *impl) GetUser(ctx context.Context, id string, clusterName *string) (*User, error) { // nolint:dupl
	sql, err := querybuilder.
		NewSelect([]querybuilder.Field{querybuilder.NewField("name")}, "system.users").
		WithCluster(clusterName).
		Where(querybuilder.WhereEquals("id", id)).
		Build()
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

	// Check if user has settings profile associated.
	{
		sql, err = querybuilder.
			NewSelect([]querybuilder.Field{querybuilder.NewField("inherit_profile")}, "system.settings_profile_elements").
			WithCluster(clusterName).
			Where(querybuilder.WhereEquals("user_name", user.Name)).
			Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}

		err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
			profile, err := data.GetNullableString("inherit_profile")
			if err != nil {
				return errors.WithMessage(err, "error scanning query result, missing 'inherit_profile' field")
			}

			user.SettingsProfile = profile
			return nil
		})
		if err != nil {
			return nil, errors.WithMessage(err, "error running query")
		}
	}

	return user, nil
}

func (i *impl) DeleteUser(ctx context.Context, id string, clusterName *string) error {
	user, err := i.GetUser(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error getting user")
	}

	if user == nil {
		// This is the desired state.
		return nil
	}

	sql, err := querybuilder.NewDropUser(user.Name).WithCluster(clusterName).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

func (i *impl) FindUserByName(ctx context.Context, name string, clusterName *string) (*User, error) {
	sql, err := querybuilder.
		NewSelect([]querybuilder.Field{querybuilder.NewField("id").ToString()}, "system.users").
		WithCluster(clusterName).
		Where(querybuilder.WhereEquals("name", name)).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var uuid string

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		uuid, err = data.GetString("id")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'id' field")
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.GetUser(ctx, uuid, clusterName)
}
