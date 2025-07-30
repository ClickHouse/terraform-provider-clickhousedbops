package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Role struct {
	ID               string   `json:"id" ch:"id"`
	Name             string   `json:"name" ch:"name"`
	SettingsProfiles []string `json:"-"`
}

func (r *Role) HasSettingProfile(profileName string) bool {
	for _, p := range r.SettingsProfiles {
		if p == profileName {
			return true
		}
	}

	return false
}

func (i *impl) CreateRole(ctx context.Context, role Role, clusterName *string) (*Role, error) {
	sql, err := querybuilder.NewCreateRole(role.Name).WithCluster(clusterName).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.FindRoleByName(ctx, role.Name, clusterName)
}

func (i *impl) GetRole(ctx context.Context, id string, clusterName *string) (*Role, error) { // nolint:dupl
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("name")},
		"system.roles",
	).WithCluster(clusterName).Where(querybuilder.WhereEquals("id", id)).Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var role *Role

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		n, err := data.GetString("name")
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

	// Check if role has settings profile associated.
	{
		sql, err = querybuilder.
			NewSelect([]querybuilder.Field{querybuilder.NewField("inherit_profile")}, "system.settings_profile_elements").
			WithCluster(clusterName).
			Where(querybuilder.WhereEquals("role_name", role.Name)).
			Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}

		profiles := make([]string, 0)
		err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
			profile, err := data.GetNullableString("inherit_profile")
			if err != nil {
				return errors.WithMessage(err, "error scanning query result, missing 'inherit_profile' field")
			}

			if profile != nil {
				profiles = append(profiles, *profile)
			}
			return nil
		})
		if err != nil {
			return nil, errors.WithMessage(err, "error running query")
		}

		role.SettingsProfiles = profiles
	}

	return role, nil
}

func (i *impl) DeleteRole(ctx context.Context, id string, clusterName *string) error {
	role, err := i.GetRole(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error getting role")
	}

	if role == nil {
		// That's what we want.
		return nil
	}

	sql, err := querybuilder.NewDropRole(role.Name).WithCluster(clusterName).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

func (i *impl) FindRoleByName(ctx context.Context, name string, clusterName *string) (*Role, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewField("id").ToString()},
		"system.roles",
	).Where(querybuilder.WhereEquals("name", name)).WithCluster(clusterName).Build()
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

	// No role with such name found.
	if uuid == "" {
		return nil, nil
	}

	return i.GetRole(ctx, uuid, clusterName)
}

func (i *impl) UpdateRole(ctx context.Context, role Role, clusterName *string) (*Role, error) {
	// Retrieve current role
	existing, err := i.GetRole(ctx, role.ID, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to get existing role")
	}

	sql, err := querybuilder.
		NewAlterRole(existing.Name).
		WithCluster(clusterName).
		RenameTo(&role.Name).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.GetRole(ctx, role.ID, clusterName)
}
