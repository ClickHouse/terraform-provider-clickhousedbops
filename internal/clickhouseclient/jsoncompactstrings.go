package clickhouseclient

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

	// Extract slice of column names in the result.
	colNames := make([]string, 0)
	for _, entry := range j.Meta {
		colNames = append(colNames, entry.Name)
	}

	// Create a slice of Rows associating the value to each column name.
	for _, row := range j.Data {
		data := Row{}

		for i, field := range row {
			data.Set(colNames[i], field)
		}

		ret = append(ret, data)
	}

	return ret
}
