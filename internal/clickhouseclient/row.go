package clickhouseclient

import (
	"fmt"
	"reflect"

	"github.com/pingcap/errors"
)

type Row struct {
	data map[string]interface{}
}

func (r *Row) GetString(fieldName string) (string, error) {
	val, ok := r.data[fieldName]
	if !ok {
		return "", errors.New(fmt.Sprintf("field %s was not found in row", fieldName))
	}

	if reflect.TypeOf(val).Name() != "string" {
		return "", errors.New(fmt.Sprintf("field %s is not a string", fieldName))
	}

	return val.(string), nil
}

func (r *Row) GetNullableString(fieldName string) (*string, error) {
	val, ok := r.data[fieldName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("field %s was not found in row", fieldName))
	}

	if reflect.TypeOf(val).String() != "*string" {
		return nil, errors.New(fmt.Sprintf("field %s is not a string pointer (%s)", fieldName, reflect.TypeOf(val).String()))
	}

	return val.(*string), nil
}

func (r *Row) GetBool(fieldName string) (bool, error) {
	val, ok := r.data[fieldName]
	if !ok {
		return false, errors.New(fmt.Sprintf("field %s was not found in row", fieldName))
	}

	switch reflect.TypeOf(val).String() {
	case "bool":
		return val.(bool), nil
	case "uint8":
		if val.(uint8) == 0 {
			return false, nil
		}
		if val.(uint8) == 1 {
			return true, nil
		}
	}

	return false, errors.New(fmt.Sprintf("unable to get field %s as bool: (%s)", fieldName, reflect.TypeOf(val).String()))
}

func (r *Row) GetUInt64(fieldName string) (uint64, error) {
	val, ok := r.data[fieldName]
	if !ok {
		return 0, errors.New(fmt.Sprintf("field %s was not found in row", fieldName))
	}

	if reflect.TypeOf(val).Name() != "uint64" {
		return 0, errors.New(fmt.Sprintf("field %s is not a uint64 (%s)", fieldName, reflect.TypeOf(val).Name()))
	}

	return val.(uint64), nil
}

func (r *Row) Set(fieldName string, val interface{}) {
	if r.data == nil {
		r.data = make(map[string]interface{})
	}
	r.data[fieldName] = val
}
