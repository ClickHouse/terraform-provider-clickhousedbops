package querybuilder

import (
	"testing"
)

func Test_selectQueryBuilder_Build(t *testing.T) {
	orderBy := NewField("col1")
	orderDirection := ASC

	tests := []struct {
		name     string
		fields   []Field
		where    []Where
		from     string
		cluster  string
		orderCol *Field
		orderDir *OrderDirection
		want     string
		wantErr  bool
	}{
		{
			name:    "Select one with",
			fields:  []Field{NewField("name")},
			from:    "users",
			want:    "SELECT `name` FROM `users`;",
			wantErr: false,
		},
		{
			name:    "Select With Cluster",
			fields:  []Field{NewField("name")},
			from:    "users",
			cluster: "cluster1",
			want:    "SELECT `name` FROM cluster('cluster1', `users`);",
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
			name:    "Select with single where",
			fields:  []Field{NewField("name")},
			where:   []Where{whereMock{"mock_where_clause"}},
			from:    "users",
			want:    "SELECT `name` FROM `users` WHERE (mock_where_clause);",
			wantErr: false,
		},
		{
			name:    "Select with multiple where",
			fields:  []Field{NewField("name")},
			where:   []Where{whereMock{"mock_where_clause"}, whereMock{"mock_where_clause_2"}},
			from:    "users",
			want:    "SELECT `name` FROM `users` WHERE (mock_where_clause AND mock_where_clause_2);",
			wantErr: false,
		},
		{
			name:     "Select with order by",
			fields:   []Field{NewField("name")},
			where:    []Where{whereMock{"mock_where_clause"}},
			orderCol: &orderBy,
			orderDir: &orderDirection,
			from:     "users",
			want:     "SELECT `name` FROM `users` WHERE (mock_where_clause) ORDER BY `col1` ASC;",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSelect(tt.fields, tt.from)
			if tt.where != nil {
				q = q.Where(tt.where...)
			}
			if tt.cluster != "" {
				q = q.WithCluster(&tt.cluster)
			}
			if tt.orderCol != nil && tt.orderDir != nil {
				q = q.OrderBy(*tt.orderCol, *tt.orderDir)
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
