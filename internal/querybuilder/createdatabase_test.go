package querybuilder

import (
	"testing"
)

func Test_createdatabase(t *testing.T) {
	comment := "this is the comment"
	clusterName := "default"
	tests := []struct {
		name         string
		action       string
		resourceType string
		resourceName string
		comment      *string
		clusterName  *string
		identified   string
		want         string
		wantErr      bool
	}{
		{
			name:         "Create database with complex name",
			resourceType: resourceTypeDatabase,
			resourceName: "data`base",
			want:         "CREATE DATABASE `data\\`base`;",
			wantErr:      false,
		},
		{
			name:         "Create database with comment",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			comment:      &comment,
			want:         "CREATE DATABASE `database` COMMENT 'this is the comment';",
			wantErr:      false,
		},
		{
			name:         "Create database with cluster",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			clusterName:  &clusterName,
			want:         "CREATE DATABASE `database` ON CLUSTER 'default';",
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q CreateDatabaseQueryBuilder
			q = &createDatabaseQueryBuilder{
				databaseName: tt.resourceName,
			}
			if tt.clusterName != nil {
				q = q.WithCluster(tt.clusterName)
			}
			if tt.comment != nil {
				q = q.WithComment(*tt.comment)
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
