package querybuilder

import (
	"fmt"
	"reflect"
)

type Where interface {
	Clause() string
}

type simpleWhere struct {
	field    string
	value    interface{}
	operator string
}

func WhereEquals(fieldName string, value interface{}) Where {
	return &simpleWhere{
		field:    fieldName,
		value:    value,
		operator: "=",
	}
}

func WhereDiffers(fieldName string, value interface{}) Where {
	return &simpleWhere{
		field:    fieldName,
		value:    value,
		operator: "<>",
	}
}

func IsNull(fieldName string) Where {
	return &simpleWhere{
		field: fieldName,
		value: nil,
	}
}

func (s *simpleWhere) Clause() string {
	if s.value == nil {
		return fmt.Sprintf("%s IS NULL", backtick(s.field))
	}

	if reflect.TypeOf(s.value).String() == "string" {
		return fmt.Sprintf("%s %s %s", backtick(s.field), s.operator, quote(s.value.(string)))
	}

	return fmt.Sprintf("%s %s %v", backtick(s.field), s.operator, s.value)
}
