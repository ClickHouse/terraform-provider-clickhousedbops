package querybuilder

import (
	"testing"
)

func Test_selectQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name    string
		fields  []Field
		options []Option
		from    string
		want    string
		wantErr bool
	}{
		{
			name:    "Select one with",
			fields:  []Field{NewField("name")},
			from:    "users",
			want:    "SELECT `name` FROM `users`;",
			wantErr: false,
		},
		{
			name:    "Select two fields",
			fields:  []Field{NewField("name"), NewField("surname")},
			from:    "users",
			want:    "SELECT `name`, `surname` FROM `users`;",
			wantErr: false,
		},
		{
			name:    "Table with database",
			fields:  []Field{NewField("name")},
			from:    "system.users",
			want:    "SELECT `name` FROM `system`.`users`;",
			wantErr: false,
		},
		{
			name:    "Select with where",
			fields:  []Field{NewField("name")},
			options: []Option{OptionMock("mock_where_clause")},
			from:    "users",
			want:    "SELECT `name` FROM `users` mock_where_clause;",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSelect(tt.fields, tt.from)
			for _, o := range tt.options {
				q = q.With(o)
			}
			got, err := q.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Build() got = %q, want %q", got, tt.want)
			}
		})
	}
}
