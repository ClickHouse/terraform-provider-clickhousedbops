package dbops

import (
	"context"
	"fmt"
	"strings"

	"github.com/pingcap/errors"
	"golang.org/x/mod/semver"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// Retrieves the ClickHouse version returns it as a CHVersion struct.
func (i *impl) GetVersion(ctx context.Context) (string, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{
			querybuilder.NewField("value"),
		},
		"system.build_options").
		Where(querybuilder.WhereEquals("name", "VERSION_FULL")).
		Build()
	if err != nil {
		return "", errors.WithMessage(err, "error building query")
	}

	var version string

	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		version, err = data.GetString("value")
		if err != nil {
			return errors.WithMessage(err, "error scanning query result")
		}
		return nil
	})
	if err != nil {
		return "", errors.WithMessage(err, "error running query")
	}

	version = strings.TrimPrefix(version, "ClickHouse ")
	parts := strings.Split(version, ".")
	if len(parts) > 3 {
		semverVersion := strings.Join(parts[:3], ".")
		version = semverVersion
	}
	version = "v" + version

	if !semver.IsValid(version) {
		return "", fmt.Errorf("invalid ClickHouse version format: %s", version)
	}

	return version, nil
}
