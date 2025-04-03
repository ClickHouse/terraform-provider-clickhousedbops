package clickhouseclient

import (
	"reflect"
	"testing"
)

func Test_jsonCompatStrings_Rows(t *testing.T) {
	tests := []struct {
		name              string
		jsonCompatStrings jsonCompatStrings
		want              []map[string]string
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
						Type: "string",
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
			want: []map[string]string{
				{
					"name": "john",
				},
				{
					"name": "frank",
				},
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
