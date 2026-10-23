package dbops

import (
	"context"
	"strings"

	"github.com/pingcap/errors"
	"golang.org/x/mod/semver"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/clickhouseclient"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// Represents the features supported by the connected ClickHouse server.
type CapabilityFlags struct {
	SourcesGrantReadWriteSeparation bool
	GrantsWildcardColumn            bool
	SourceGrantFilters              bool
}

// Initialize a new CapabilityFlags structure based on the ClickHouse version.
func NewCapabilityFlags(chVersion string) *CapabilityFlags {
	flags := &CapabilityFlags{}

	if semver.Compare(chVersion, "v25.7.0") >= 0 {
		flags.SourcesGrantReadWriteSeparation = true
	}

	if semver.Compare(chVersion, "v26.7.0") >= 0 {
		flags.GrantsWildcardColumn = true
	}

	return flags
}

// detectCapabilityFlags adds the features gated by server settings to the version-gated ones.
func (i *impl) detectCapabilityFlags(ctx context.Context) (*CapabilityFlags, error) {
	version, err := i.GetVersion(ctx)
	if err != nil {
		i.initErr = err
		return nil, err
	}

	// Source filters show up in system.grants since 26.3, and only with enable_read_write_grants.
	sourceGrantFilters := false
	if semver.Compare(version, "v26.3.0") >= 0 {
		var err error
		if sourceGrantFilters, err = i.readWriteGrantsEnabled(ctx); err != nil {
			return nil, err
		}
	}

	return &CapabilityFlags{
		SourcesGrantReadWriteSeparation: semver.Compare(version, "v25.7.0") >= 0,
		GrantsWildcardColumn:            semver.Compare(version, "v26.7.0") >= 0,
		SourceGrantFilters:              sourceGrantFilters,
	}, nil
}

// readWriteGrantsEnabled detects enable_read_write_grants: without it, ClickHouse rewrites READ ON URL to the legacy URL ON *.*.
func (i *impl) readWriteGrantsEnabled(ctx context.Context) (bool, error) {
	sql, err := querybuilder.NewSelect(
		[]querybuilder.Field{querybuilder.NewRawField("formatQuerySingleLine('GRANT READ ON URL TO r')", "formatted")},
		"system.one",
	).Build()
	if err != nil {
		return false, errors.WithMessage(err, "error building query")
	}

	var formatted string
	err = i.clickhouseClient.Select(ctx, sql, func(data clickhouseclient.Row) error {
		formatted, err = data.GetString("formatted")
		return err
	})
	if err != nil {
		return false, errors.WithMessage(err, "error checking access_control_improvements.enable_read_write_grants")
	}

	return strings.HasPrefix(formatted, "GRANT READ"), nil
}
