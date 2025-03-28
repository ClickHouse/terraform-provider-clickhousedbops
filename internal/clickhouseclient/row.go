package clickhouseclient

import (
	"fmt"

	"github.com/pingcap/errors"
)

type Row struct {
	data map[string]string
}

func (r *Row) Get(fieldName string) (string, error) {
	val, ok := r.data[fieldName]
	if !ok {
		return "", errors.New(fmt.Sprintf("field %s was not found in row", fieldName))
	}

	return val, nil
}

func (r *Row) Set(fieldName string, val string) {
	if r.data == nil {
		r.data = make(map[string]string)
	}
	r.data[fieldName] = val
}
