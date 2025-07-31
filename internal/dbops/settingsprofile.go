package dbops

import (
	"context"
	"fmt"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type SettingsProfile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	InheritFrom []string `json:"-"`
}

func (i *impl) CreateSettingsProfile(ctx context.Context, profile SettingsProfile, clusterName *string) (*SettingsProfile, error) {
	sql, err := querybuilder.
		NewCreateSettingsProfile(profile.Name).
		WithCluster(clusterName).
		InheritFrom(profile.InheritFrom).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.FindSettingsProfileByName(ctx, profile.Name, clusterName)
}

func (i *impl) GetSettingsProfile(ctx context.Context, id string, clusterName *string) (*SettingsProfile, error) {
	var profile *SettingsProfile

	sql, err := querybuilder.
		NewSelect(
			[]querybuilder.Field{
				querybuilder.NewField("name"),
			},
			"system.settings_profiles",
		).
		WithCluster(clusterName).
		Where(querybuilder.WhereEquals("id", id)).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		name, err := data.GetString("name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'name' field")
		}

		if profile == nil {
			profile = &SettingsProfile{
				ID:   id,
				Name: name,
			}
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if profile == nil {
		// SettingsProfile not found
		return nil, nil
	}

	// Check roles this profile is inheriting from.
	{
		sql, err := querybuilder.
			NewSelect([]querybuilder.Field{querybuilder.NewField("inherit_profile")}, "system.settings_profile_elements").
			Where(querybuilder.WhereEquals("profile_name", profile.Name)).
			OrderBy(querybuilder.NewField("index"), querybuilder.ASC).
			Build()
		if err != nil {
			return nil, errors.WithMessage(err, "error building query")
		}
		err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
			inheritedProfileName, err := data.GetNullableString("inherit_profile")
			if err != nil {
				return errors.WithMessage(err, "error scanning query result, missing 'profile_name' field")
			}

			if inheritedProfileName != nil {
				profile.InheritFrom = append(profile.InheritFrom, *inheritedProfileName)
			}

			return nil
		})
		if err != nil {
			return nil, errors.WithMessage(err, "error running query")
		}
	}

	return profile, nil
}

func (i *impl) DeleteSettingsProfile(ctx context.Context, id string, clusterName *string) error {
	profile, err := i.GetSettingsProfile(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error looking up settings profile name")
	}

	if profile == nil {
		// Desired status
		return nil
	}

	sql, err := querybuilder.NewDropSettingsProfile(profile.Name).WithCluster(clusterName).Build()
	if err != nil {
		return errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return errors.WithMessage(err, "error running query")
	}

	return nil
}

func (i *impl) UpdateSettingsProfile(ctx context.Context, settingsProfile SettingsProfile, clusterName *string) (*SettingsProfile, error) {
	// Retrieve current setting profile
	existing, err := i.GetSettingsProfile(ctx, settingsProfile.ID, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "Unable to get existing settings profile")
	}

	if existing == nil {
		return nil, nil
	}

	sql, err := querybuilder.
		NewAlterSettingsProfile(existing.Name).
		WithCluster(clusterName).
		InheritFrom(settingsProfile.InheritFrom).
		RenameTo(&settingsProfile.Name).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.GetSettingsProfile(ctx, settingsProfile.ID, clusterName)
}

func (i *impl) AssociateSettingsProfile(ctx context.Context, id string, roleId *string, userId *string, clusterName *string) error {
	profile, err := i.GetSettingsProfile(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error looking up settings profile name")
	}

	if profile == nil {
		return errors.New("No Settings Profile with such ID found")
	}

	if roleId != nil {
		role, err := i.GetRole(ctx, *roleId, clusterName)
		if err != nil {
			return errors.WithMessage(err, "Cannot find role")
		}

		if role == nil {
			return errors.New("role not found")
		}
		sql, err := querybuilder.
			NewAlterRole(role.Name).
			WithCluster(clusterName).
			AddSettingsProfile(&profile.Name).
			Build()
		if err != nil {
			return errors.WithMessage(err, "Error building query")
		}

		err = i.clickhouseClient.Exec(ctx, sql)
		if err != nil {
			return errors.WithMessage(err, "error running query")
		}

		return nil
	} else if userId != nil {
		user, err := i.GetUser(ctx, *userId, clusterName)
		if err != nil {
			return errors.WithMessage(err, "Cannot find user")
		}

		if user == nil {
			return errors.New("user not found")
		}

		sql, err := querybuilder.
			NewAlterUser(user.Name).
			WithCluster(clusterName).
			AddSettingsProfile(&profile.Name).
			Build()
		if err != nil {
			return errors.WithMessage(err, "Error building query")
		}

		err = i.clickhouseClient.Exec(ctx, sql)
		if err != nil {
			return errors.WithMessage(err, "error running query")
		}

		return nil
	}

	return errors.New("Neither roleId nor userId were specified")
}

func (i *impl) DisassociateSettingsProfile(ctx context.Context, id string, roleId *string, userId *string, clusterName *string) error {
	profile, err := i.GetSettingsProfile(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error looking up settings profile name")
	}

	if profile == nil {
		return errors.New("No Settings Profile with such ID found")
	}

	if roleId != nil {
		role, err := i.GetRole(ctx, *roleId, clusterName)
		if err != nil {
			return errors.WithMessage(err, "Cannot find role")
		}

		if role == nil {
			return errors.New("role not found")
		}

		sql, err := querybuilder.
			NewAlterRole(role.Name).
			WithCluster(clusterName).
			DropSettingsProfile(&profile.Name).
			Build()
		if err != nil {
			return errors.WithMessage(err, "Error building query")
		}

		err = i.clickhouseClient.Exec(ctx, sql)
		if err != nil {
			return errors.WithMessage(err, "error running query")
		}

		return nil
	} else if userId != nil {
		user, err := i.GetUser(ctx, *userId, clusterName)
		if err != nil {
			return errors.WithMessage(err, "Cannot find user")
		}

		if user == nil {
			return errors.New("user not found")
		}

		sql, err := querybuilder.
			NewAlterUser(user.Name).
			WithCluster(clusterName).
			DropSettingsProfile(&profile.Name).
			Build()
		if err != nil {
			return errors.WithMessage(err, "Error building query")
		}

		err = i.clickhouseClient.Exec(ctx, sql)
		if err != nil {
			return errors.WithMessage(err, "error running query")
		}

		return nil
	}

	return errors.New("Neither roleId nor userId were specified")
}

func (i *impl) FindSettingsProfileByName(ctx context.Context, name string, clusterName *string) (*SettingsProfile, error) {
	sql, err := querybuilder.
		NewSelect(
			[]querybuilder.Field{
				querybuilder.NewField("id").ToString(),
			},
			"system.settings_profiles",
		).
		WithCluster(clusterName).
		Where(querybuilder.WhereEquals("name", name)).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var settingsProfileID string

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		id, err := data.GetString("id")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'id' field")
		}

		settingsProfileID = id

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	if settingsProfileID == "" {
		return nil, errors.New(fmt.Sprintf("settings profile with name %s not found", name))
	}

	return i.GetSettingsProfile(ctx, settingsProfileID, clusterName)
}
