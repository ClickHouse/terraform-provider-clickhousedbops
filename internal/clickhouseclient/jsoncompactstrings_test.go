package clickhouseclient

import (
	"reflect"
	"testing"
)

func Test_jsonCompatStrings_Rows(t *testing.T) {
	tests := []struct {
		name              string
		jsonCompatStrings jsonCompatStrings
		want              []Row
	}{
		{
			name: "Basic test",
			jsonCompatStrings: jsonCompatStrings{
				Meta: []struct {
					Name string
					Type string
				}{
					{
						Name: "name",
						Type: "String",
					},
				},
				Data: [][]string{
					{
						"john",
					},
					{
						"frank",
					},
				},
			},
			want: []Row{
				rowFromMap(map[string]string{
					"name": "john",
				}),
				rowFromMap(map[string]string{
					"name": "frank",
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.jsonCompatStrings.Rows()

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jsonCompatStrings.Rows() want = %v, got %v", tt.want, got)
			}
		})
	}
}

func rowFromMap(data map[string]string) Row {
	row := Row{}

	for k, v := range data {
		row.Set(k, v)
	}

	return row
}
