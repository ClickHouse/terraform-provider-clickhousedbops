package querybuilder

// QueryBuilder is an interface meant to build SQL queries (already interpolated) with pluggable options.
type QueryBuilder interface {
	Build() (string, error)
}
