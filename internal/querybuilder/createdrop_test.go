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
		options      []Option
		want         string
		wantErr      bool
	}{
		{
			name:         "Create database with complex name",
			action:       actionCreate,
			resourceType: resourceTypeDatabase,
			resourceName: "data`base",
			want:         "CREATE DATABASE `data\\`base`;",
			wantErr:      false,
		},
		{
			name:         "Create database with comment",
			action:       actionCreate,
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			options:      []Option{Comment("this is the comment")},
			want:         "CREATE DATABASE `database` COMMENT 'this is the comment';",
			wantErr:      false,
		},
		{
			name:         "Drop database",
			action:       actionDrop,
			resourceType: resourceTypeDatabase,
			resourceName: "db1",
			options:      nil,
			want:         "DROP DATABASE `db1`;",
			wantErr:      false,
		},
		{
			name:         "Drop database with complex name",
			action:       actionDrop,
			resourceType: resourceTypeDatabase,
			resourceName: "data`base",
			options:      nil,
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := createDropQueryBuilder{
				action:           tt.action,
				resourceTypeName: tt.resourceType,
				resourceName:     tt.resourceName,
				options:          tt.options,
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
