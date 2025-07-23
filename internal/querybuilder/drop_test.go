package querybuilder

import (
	"testing"
)

func Test_drop(t *testing.T) {
	cluster := "cluster1"

	tests := []struct {
		name         string
		action       string
		resourceType string
		resourceName string
		comment      string
		identified   string
		clusterName  *string
		want         string
		wantErr      bool
	}{
		{
			name:         "Drop database",
			resourceType: resourceTypeDatabase,
			resourceName: "db1",
			want:         "DROP DATABASE `db1`;",
			wantErr:      false,
		},
		{
			name:         "Drop database on cluster",
			resourceType: resourceTypeDatabase,
			resourceName: "db1",
			clusterName:  &cluster,
			want:         "DROP DATABASE `db1` ON CLUSTER 'cluster1';",
			wantErr:      false,
		},
		{
			name:         "Drop database with complex name",
			resourceType: resourceTypeDatabase,
			resourceName: "data`base",
			want:         "DROP DATABASE `data\\`base`;",
			wantErr:      false,
		},
		{
			name:         "Fail to create role with empty name",
			resourceType: resourceTypeRole,
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "Drop role with simple name",
			resourceType: resourceTypeRole,
			resourceName: "role1",
			want:         "DROP ROLE `role1`;",
			wantErr:      false,
		},
		{
			name:         "Drop role with complex name",
			resourceType: resourceTypeRole,
			resourceName: "ro`le1",
			want:         "DROP ROLE `ro\\`le1`;",
			wantErr:      false,
		},
		{
			name:         "Fail to drop role with empty name",
			resourceType: resourceTypeRole,
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
		{
			name:         "Drop user with simple name",
			resourceType: resourceTypeUser,
			resourceName: "john",
			want:         "DROP USER `john`;",
			wantErr:      false,
		},
		{
			name:         "Drop user with complex name",
			resourceType: resourceTypeUser,
			resourceName: "jo`hn",
			want:         "DROP USER `jo\\`hn`;",
			wantErr:      false,
		},
		{
			name:         "Fail to drop user with empty name",
			resourceType: resourceTypeUser,
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := dropQueryBuilder{
				resourceTypeName: tt.resourceType,
				resourceName:     tt.resourceName,
				clusterName:      tt.clusterName,
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
