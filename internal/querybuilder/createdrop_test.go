package querybuilder

import (
	"testing"
)

func Test_create_drop(t *testing.T) {
	tests := []struct {
		name         string
		action       string
		resourceType string
		resourceName string
		comment      string
		identified   string
		want         string
		wantErr      bool
	}{
		{
			name:         "Drop database",
			action:       actionDrop,
			resourceType: resourceTypeDatabase,
			resourceName: "db1",
			want:         "DROP DATABASE `db1`;",
			wantErr:      false,
		},
		{
			name:         "Drop database with complex name",
			action:       actionDrop,
			resourceType: resourceTypeDatabase,
			resourceName: "data`base",
			want:         "DROP DATABASE `data\\`base`;",
			wantErr:      false,
		},
		{
			name:         "Create role with simple name",
			action:       actionCreate,
			resourceType: resourceTypeRole,
			resourceName: "role1",
			want:         "CREATE ROLE `role1`;",
			wantErr:      false,
		},
		{
			name:         "Create role with complex name",
			action:       actionCreate,
			resourceType: resourceTypeRole,
			resourceName: "ro`le1",
			want:         "CREATE ROLE `ro\\`le1`;",
			wantErr:      false,
		},
		{
			name:         "Fail to create role with empty name",
			action:       actionCreate,
			resourceType: resourceTypeRole,
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "Drop role with simple name",
			action:       actionDrop,
			resourceType: resourceTypeRole,
			resourceName: "role1",
			want:         "DROP ROLE `role1`;",
			wantErr:      false,
		},
		{
			name:         "Drop role with complex name",
			action:       actionDrop,
			resourceType: resourceTypeRole,
			resourceName: "ro`le1",
			want:         "DROP ROLE `ro\\`le1`;",
			wantErr:      false,
		},
		{
			name:         "Fail to drop role with empty name",
			action:       actionDrop,
			resourceType: resourceTypeRole,
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "Drop user with simple name",
			action:       actionDrop,
			resourceType: resourceTypeUser,
			resourceName: "john",
			want:         "DROP USER `john`;",
			wantErr:      false,
		},
		{
			name:         "Drop user with complex name",
			action:       actionDrop,
			resourceType: resourceTypeUser,
			resourceName: "jo`hn",
			want:         "DROP USER `jo\\`hn`;",
			wantErr:      false,
		},
		{
			name:         "Fail to drop user with empty name",
			action:       actionDrop,
			resourceType: resourceTypeUser,
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := createDropQueryBuilder{
				action:           tt.action,
				resourceTypeName: tt.resourceType,
				resourceName:     tt.resourceName,
			}

			got, err := q.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Build() got = %v, want %v", got, tt.want)
			}
		})
	}
}
