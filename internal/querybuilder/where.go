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

func IsNull(fieldName string) Where {
	return &simpleWhere{
		field: fieldName,
		value: nil,
	}
}

type simpleWhere struct {
	field string
	value interface{}
}

func (s *simpleWhere) Clause() string {
	if s.value == nil {
		return fmt.Sprintf("%s IS NULL", backtick(s.field))
	}

	if reflect.TypeOf(s.value).String() == "string" {
		return fmt.Sprintf("%s = %s", backtick(s.field), quote(s.value.(string)))
	}

	return fmt.Sprintf("%s = %v", backtick(s.field), s.value)
}
