package dbops

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
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

func (i *impl) CreateSetting(ctx context.Context, settingsProfileID string, setting Setting, clusterName *string, timeout time.Duration) (*Setting, error) {
	settingsProfile, err := i.GetSettingsProfile(ctx, settingsProfileID, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting settings profile")
	}

	if settingsProfile == nil {
		return nil, errors.New(fmt.Sprintf("settings profile with id %q was not found", settingsProfileID))
	}

	sql, err := querybuilder.NewAlterSettingsProfile(settingsProfile.Name).
		WithCluster(clusterName).
		AddSetting(setting.Name, setting.Value, setting.Min, setting.Max, setting.Writability).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	err = i.clickhouseClient.Exec(ctx, sql)
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	// Retry to handle potential replication lag
	retryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	backoff := 50 * time.Millisecond
	for {
		createdSetting, err := i.GetSetting(retryCtx, settingsProfileID, setting.Name, clusterName)
		if err != nil {
			return nil, errors.WithMessage(err, "error retrieving created setting")
		}

		if createdSetting != nil {
			return createdSetting, nil
		}

		tflog.Debug(ctx, "Setting not found, retrying with exponential backoff", map[string]any{
			"setting_name": setting.Name,
			"backoff":      backoff.String(),
		})

		// Context-aware sleep with exponential backoff
		timer := time.NewTimer(backoff)
		defer timer.Stop()

		select {
		case <-retryCtx.Done():
			return nil, fmt.Errorf(
				"setting %q was created but could not be retrieved within timeout (%v): %w",
				setting.Name, timeout, retryCtx.Err(),
			)
		case <-timer.C:
			backoff *= 2
		}
	}
}

func (i *impl) GetSetting(ctx context.Context, settingsProfileID string, name string, clusterName *string) (*Setting, error) {
	settingsProfile, err := i.GetSettingsProfile(ctx, settingsProfileID, clusterName)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting settings profile")
	}

	if settingsProfile == nil {
		// No setting profile, hence no setting available.
		return nil, nil
	}

	sql, err := querybuilder.NewSelect([]querybuilder.Field{
		querybuilder.NewField("value"),
		querybuilder.NewField("min"),
		querybuilder.NewField("max"),
		querybuilder.NewField("writability").ToString(),
	}, "system.settings_profile_elements").
		WithCluster(clusterName).
		Where(querybuilder.AndWhere(
			querybuilder.WhereEquals("profile_name", settingsProfile.Name),
			querybuilder.WhereEquals("setting_name", name),
		)).
		Build()
	if err != nil {
		return nil, errors.WithMessage(err, "error building query")
	}

	var setting *Setting

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		value, err := data.GetNullableString("value")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'value' field")
		}

		minV, err := data.GetNullableString("min")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'min' field")
		}

		maxV, err := data.GetNullableString("max")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'max' field")
		}

		writability, err := data.GetNullableString("writability")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result, missing 'writability' field")
		}

		if setting == nil {
			setting = &Setting{
				Name:        name,
				Value:       value,
				Min:         minV,
				Max:         maxV,
				Writability: writability,
			}
		}

		return nil
	})
	if err != nil {
		return nil, errors.WithMessage(err, "error running query")
	}

	return setting, nil
}

func (i *impl) DeleteSetting(ctx context.Context, settingsProfileID string, name string, clusterName *string) error {
	settingsProfile, err := i.GetSettingsProfile(ctx, settingsProfileID, clusterName)
	if err != nil {
		return errors.WithMessage(err, "error getting settings profile")
	}

	if settingsProfile == nil {
		return errors.New(fmt.Sprintf("settings profile with id %q was not found", settingsProfileID))
	}

	sql, err := querybuilder.NewAlterSettingsProfile(settingsProfile.Name).
		WithCluster(clusterName).
		RemoveSetting(name).
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
