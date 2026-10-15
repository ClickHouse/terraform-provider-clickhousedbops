package dbops

import (
	"golang.org/x/mod/semver"
)

// Represents the features supported by the connected ClickHouse server.
type CapabilityFlags struct {
	SourcesGrantReadWriteSeparation bool
	GrantsWildcardColumn            bool
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
