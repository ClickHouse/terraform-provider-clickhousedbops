package dbops

import (
	"context"
	"slices"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type SettingsProfile struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	InheritFrom   []string `json:"-"`
	ApplyToAll    bool     `json:"-"`
	ApplyToList   []string `json:"-"`
	ApplyToExcept []string `json:"-"`
}

func (p *SettingsProfile) AppliesTo(name string) bool {
	return slices.Contains(p.ApplyToList, name)
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
		if !isAlreadyExistsError(err) {
			return nil, errors.WithMessage(err, "error running query")
		}

		// The settings profile already exists in ClickHouse, importing it instead of failing.
		tflog.Warn(ctx, "settings profile already exists, adopting it instead of failing", map[string]any{
			"name": profile.Name,
		})
	}

	return retryWithBackoff(ctx, "settings profile", profile.Name, func() (*SettingsProfile, error) {
		return i.FindSettingsProfileByName(ctx, profile.Name, clusterName)
	}, i.readAfterWriteTimeoutArgs()...)
}

func (i *impl) GetSettingsProfile(ctx context.Context, id string, clusterName *string) (*SettingsProfile, error) {
	var profile *SettingsProfile

	sql, err := querybuilder.
		NewSelect(
			[]querybuilder.Field{
				querybuilder.NewField("name"),
				querybuilder.NewField("apply_to_all"),
				querybuilder.NewRawField("arrayStringConcat(apply_to_list, '\\n')", "apply_to_list"),
				querybuilder.NewRawField("arrayStringConcat(apply_to_except, '\\n')", "apply_to_except"),
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

		applyToAll, err := data.GetBool("apply_to_all")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'apply_to_all' field")
		}

		applyToList, err := data.GetString("apply_to_list")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'apply_to_list' field")
		}

		applyToExcept, err := data.GetString("apply_to_except")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'apply_to_except' field")
		}

		if profile == nil {
			profile = &SettingsProfile{
				ID:            id,
				Name:          name,
				ApplyToAll:    applyToAll,
				ApplyToList:   splitNonEmpty(applyToList),
				ApplyToExcept: splitNonEmpty(applyToExcept),
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

// AddSettingsProfileApplyTo adds a user or role to the profile's TO list with ALTER SETTINGS PROFILE, for users that
// ALTER USER cannot change, such as JWT ephemeral users. ClickHouse has no ADD TO, so the list is read and rewritten.
func (i *impl) AddSettingsProfileApplyTo(ctx context.Context, id string, name string, clusterName *string) error {
	i.applyToMu.Lock()
	defer i.applyToMu.Unlock()

	profile, err := i.GetSettingsProfile(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error looking up settings profile")
	}

	if profile == nil {
		return errors.New("No Settings Profile with such ID found")
	}

	if profile.ApplyToAll {
		return errors.Errorf("settings profile %q applies to ALL, refusing to rewrite its TO list", profile.Name)
	}

	if profile.AppliesTo(name) {
		return nil
	}

	return i.setSettingsProfileApplyTo(ctx, profile.Name, append(slices.Clone(profile.ApplyToList), name), clusterName)
}

// RemoveSettingsProfileApplyTo removes a user or role from the profile's TO list and keeps the other entries.
func (i *impl) RemoveSettingsProfileApplyTo(ctx context.Context, id string, name string, clusterName *string) error {
	i.applyToMu.Lock()
	defer i.applyToMu.Unlock()

	profile, err := i.GetSettingsProfile(ctx, id, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error looking up settings profile")
	}

	if profile == nil || !profile.AppliesTo(name) {
		// ClickHouse lists only the entries it can resolve to a name: a user that does not exist right now cannot be
		// named in a TO clause, so it is left as is.
		return nil
	}

	remaining := slices.DeleteFunc(slices.Clone(profile.ApplyToList), func(n string) bool { return n == name })

	return i.setSettingsProfileApplyTo(ctx, profile.Name, remaining, clusterName)
}

func (i *impl) setSettingsProfileApplyTo(ctx context.Context, profileName string, names []string, clusterName *string) error {
	sql, err := querybuilder.
		NewAlterSettingsProfile(profileName).
		WithCluster(clusterName).
		ApplyTo(names).
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
		// Not found: (nil, nil) keeps retryWithBackoff callers retrying.
		return nil, nil
	}

	return i.GetSettingsProfile(ctx, settingsProfileID, clusterName)
}
