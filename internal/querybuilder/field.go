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
