package querybuilder

import (
	"fmt"
	"reflect"
)

func Where(fieldName string, value interface{}) Option {
	return &simpleWhere{
		field: fieldName,
		value: value,
	}
}

type simpleWhere struct {
	field string
	value interface{}
}

func (s *simpleWhere) String() string {
	if reflect.TypeOf(s.value).String() == "string" {
		return fmt.Sprintf("WHERE %s = %s", backtick(s.field), quote(s.value.(string)))
	}

	return fmt.Sprintf("WHERE %s = %v", backtick(s.field), s.value)
}
