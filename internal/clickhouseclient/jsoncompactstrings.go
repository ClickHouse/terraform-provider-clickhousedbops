package clickhouseclient

type jsonCompatStrings struct {
	Meta []struct {
		Name string
		Type string
	} `json:"meta"`
	Data [][]string `json:"data"`
}

func (j jsonCompatStrings) Rows() []Row {
	ret := make([]Row, 0)

	colNames := make([]string, 0)
	for _, entry := range j.Meta {
		colNames = append(colNames, entry.Name)
	}

	for _, row := range j.Data {
		data := Row{}

		for i, field := range row {
			data.Set(colNames[i], field)
		}

		ret = append(ret, data)
	}

	return ret
}
