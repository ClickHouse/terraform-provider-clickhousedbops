package dbops

import (
	"context"

	"github.com/pingcap/errors"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

type Setting struct {
	Name        string
	Value       *string
	Min         *string
	Max         *string
	Writability *string
}

type SettingsProfile struct {
	Name string `json:"name"`

	InheritProfile *string
	Settings       []Setting
}

func (i *impl) CreateSettingsProfile(ctx context.Context, profile SettingsProfile, clusterName *string) (*SettingsProfile, error) {
	builder := querybuilder.
		NewCreateSettingsProfile(profile.Name).
		WithCluster(clusterName).
		WithInheritProfile(profile.InheritProfile)

	for _, setting := range profile.Settings {
		builder.AddSetting(setting.Name, setting.Value, setting.Min, setting.Max, setting.Writability)
	}

	sql, err := builder.Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return i.GetSettingsProfile(ctx, profile.Name, clusterName)
}

func (i *impl) GetSettingsProfile(ctx context.Context, name string, clusterName *string) (*SettingsProfile, error) {
	sql, err := querybuilder.
		NewSelect(
			[]querybuilder.Field{
				querybuilder.NewField("profile_name"),
				querybuilder.NewField("setting_name"),
				querybuilder.NewField("value"),
				querybuilder.NewField("min"),
				querybuilder.NewField("max"),
				querybuilder.NewField("writability").ToString(),
				querybuilder.NewField("inherit_profile"),
			},
			"system.settings_profile_elements",
		).
		WithCluster(clusterName).
		Where(querybuilder.WhereEquals("profile_name", name)).
		OrderBy(querybuilder.NewField("index"), querybuilder.ASC).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var profile *SettingsProfile

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		n, err := data.GetNullableString("profile_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'profile_name' field")
		}

		settingName, err := data.GetNullableString("setting_name")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'setting_name' field")
		}

		value, err := data.GetNullableString("value")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'value' field")
		}

		minVal, err := data.GetNullableString("min")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'min' field")
		}

		maxVal, err := data.GetNullableString("max")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'max' field")
		}

		writability, err := data.GetNullableString("writability")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'writability' field")
		}

		inherit, err := data.GetNullableString("inherit_profile")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'inherit_profile' field")
		}

		if profile == nil {
			profile = &SettingsProfile{
				Name:     *n,
				Settings: make([]Setting, 0),
			}
		}

		if inherit != nil {
			profile.InheritProfile = inherit
		}

		if settingName != nil {
			profile.Settings = append(profile.Settings, Setting{
				Name:        *settingName,
				Value:       value,
				Min:         minVal,
				Max:         maxVal,
				Writability: writability,
			})
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

	return profile, nil
}

func (i *impl) DeleteSettingsProfile(ctx context.Context, name string, clusterName *string) error {
	profile, err := i.GetSettingsProfile(ctx, name, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error getting database name")
	}

	if profile == nil {
		// This is desired state.
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

func (i *impl) AssociateSettingsProfile(ctx context.Context, name string, roleId *string, userId *string, clusterName *string) error {
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
			AddSettingsProfile(&name).
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
			AddSettingsProfile(&name).
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

func (i *impl) DisassociateSettingsProfile(ctx context.Context, name string, roleId *string, userId *string, clusterName *string) error {
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
			DropSettingsProfile(&name).
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
			DropSettingsProfile(&name).
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
