package querybuilder

// QueryBuilder is an interface meant to build SQL queries (already interpolated) with pluggable options.
type QueryBuilder interface {
	Build() (string, error)
}

// HiddenValue replaces secrets in masked queries. ClickHouse uses the same placeholder
// when it masks named collection statements in system.query_log.
const HiddenValue = "[HIDDEN]"

// MaskedQueryBuilder is implemented by builders interpolating values that must not be logged.
type MaskedQueryBuilder interface {
	QueryBuilder
	BuildMasked() (string, error)
}
