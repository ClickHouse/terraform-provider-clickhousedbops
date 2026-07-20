package grants

import (
	"slices"
	"strings"
)

// Grant is a privilege grant reduced to the fields that determine coverage.
type Grant struct {
	AccessType   string
	Database     *string
	Table        *string
	Column       *string
	AccessObject *string
	GrantOption  bool
}

// Covers reports whether broader already conveys at least narrower. Both are
// assumed to target the same grantee.
func Covers(broader, narrower Grant) bool {
	// broader must be narrower's privilege, or a group that contains it.
	if !slices.Contains(AllDescendants(Parsed().Groups, broader.AccessType), narrower.AccessType) {
		return false
	}
	// A grant that needs grant option is not covered by one lacking it.
	if narrower.GrantOption && !broader.GrantOption {
		return false
	}
	return objectCovers(broader.Database, narrower.Database) &&
		objectCovers(broader.Table, narrower.Table) &&
		objectCovers(broader.Column, narrower.Column) &&
		objectCovers(broader.AccessObject, narrower.AccessObject)
}

// objectCovers reports whether broader covers narrower on a single dimension; nil means wildcard.
// ClickHouse does not return the trailing '*' of a prefix grant, so a stored value cannot be
// distinguished from a wildcard and is always treated as a prefix.
func objectCovers(broader, narrower *string) bool {
	if broader == nil {
		return true
	}
	if narrower == nil {
		return false
	}
	if *broader == *narrower {
		return true
	}
	return strings.HasPrefix(strings.TrimSuffix(*narrower, "*"), strings.TrimSuffix(*broader, "*"))
}
