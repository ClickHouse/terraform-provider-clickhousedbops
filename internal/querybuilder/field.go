package querybuilder

import (
	"fmt"
)

type Field interface {
	ToString() Field
	SQLDef() string
}

type field struct {
	name     string
	toString bool
}

func NewField(name string) Field {
	return &field{
		name: name,
	}
}

func (f *field) ToString() Field {
	f.toString = true
	return f
}

func (f *field) SQLDef() string {
	if f.toString {
		return fmt.Sprintf("toString(%s) AS %s", backtick(f.name), backtick(f.name))
	}
	return backtick(f.name)
}

// rawField emits a verbatim SQL expression aliased to a column name, for computed reads
// (e.g. arrayStringConcat) that NewField's identifier escaping cannot express.
type rawField struct {
	expr  string
	alias string
}

// NewRawField returns a Field that emits `<expr> AS <alias>`. expr is used verbatim (the caller
// owns its correctness); alias is backtick-escaped and becomes the result column name.
func NewRawField(expr string, alias string) Field {
	return &rawField{expr: expr, alias: alias}
}

func (f *rawField) ToString() Field {
	return f
}

func (f *rawField) SQLDef() string {
	return fmt.Sprintf("%s AS %s", f.expr, backtick(f.alias))
}
