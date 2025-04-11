package querybuilder

import (
	"fmt"
	"reflect"
)

type Where interface {
	Clause() string
}

func SimpleWhere(fieldName string, value interface{}) Where {
	return &simpleWhere{
		field: fieldName,
		value: value,
	}
}

type simpleWhere struct {
	field string
	value interface{}
}

func (s *simpleWhere) Clause() string {
	if reflect.TypeOf(s.value).String() == "string" {
		return fmt.Sprintf("WHERE %s = %s", backtick(s.field), quote(s.value.(string)))
	}

	return fmt.Sprintf("WHERE %s = %v", backtick(s.field), s.value)
}
