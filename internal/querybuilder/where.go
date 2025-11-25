package querybuilder

import (
	"fmt"
	"reflect"
	"strings"
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

func WhereIn(fieldName string, value interface{}) Where {
	return &simpleWhere{
		field:    fieldName,
		value:    value,
		operator: "IN",
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

	if reflect.TypeOf(s.value).Kind() == reflect.Slice {
		sliceValue := reflect.ValueOf(s.value)
		values := make([]string, sliceValue.Len())

		elemType := reflect.TypeOf(s.value).Elem()
		isStringSlice := elemType.Kind() == reflect.String

		for i := 0; i < sliceValue.Len(); i++ {
			elem := sliceValue.Index(i).Interface()
			if isStringSlice {
				values[i] = quote(elem.(string))
			} else {
				values[i] = fmt.Sprintf("%v", elem)
			}
		}

		return fmt.Sprintf("%s %s (%s)", backtick(s.field), s.operator, strings.Join(values, ", "))
	}

	return fmt.Sprintf("%s %s %v", backtick(s.field), s.operator, s.value)
}
