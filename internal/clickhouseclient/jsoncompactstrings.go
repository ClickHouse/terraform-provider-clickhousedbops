package clickhouseclient

import (
	"fmt"
	"strconv"
)

const nullString = "ᴺᵁᴸᴸ"

// jsonCompatStrings is used to parse clickhouse query output when using 'jsonCompatStrings' format.
type jsonCompatStrings struct {
	Meta []struct {
		Name string
		Type string
	} `json:"meta"`
	Data [][]string `json:"data"`
}

func (j jsonCompatStrings) Rows() []Row {
	ret := make([]Row, 0)

	// Extract slice of column names and column data types in the result.
	colNames := make([]string, 0)
	colTypes := make([]string, 0)
	for _, entry := range j.Meta {
		colNames = append(colNames, entry.Name)
		colTypes = append(colTypes, entry.Type)
	}

	// Create a slice of Rows associating the value to each column name.
	for _, row := range j.Data {
		data := Row{}

		for i, field := range row {
			switch colTypes[i] {
			case "String":
				data.Set(colNames[i], field)
			case "Nullable(String)":
				if field == nullString {
					data.Set(colNames[i], nilPtr[string]())
				} else {
					data.Set(colNames[i], &field)
				}
			case "UInt8":
				val, err := strconv.ParseUint(field, 10, 8)
				if err != nil {
					// Failed parsing as number, return value as-is.
					data.Set(colNames[i], field)
					break
				} else {
					data.Set(colNames[i], uint8(val))
				}
			case "UInt64":
				val, err := strconv.ParseUint(field, 10, 64)
				if err != nil {
					// Failed parsing as number, return value as-is.
					data.Set(colNames[i], field)
					break
				} else {
					data.Set(colNames[i], val)
				}
			default:
				panic(fmt.Sprintf("unknown data type %q", colTypes[i]))
			}
		}

		ret = append(ret, data)
	}

	return ret
}

func nilPtr[T any]() *T {
	var r *T
	return r
}
