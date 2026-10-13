package querybuilder

import (
	"testing"
)

func Test_createdatabase(t *testing.T) {
	tests := []struct {
		name         string
		action       string
		resourceType string
		resourceName string
		comment      *string
		engine       *string
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
			comment:      new("this is the comment"),
			want:         "CREATE DATABASE `database` COMMENT 'this is the comment';",
			wantErr:      false,
		},
		{
			name:         "Create database with cluster",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			clusterName:  new("default"),
			want:         "CREATE DATABASE `database` ON CLUSTER 'default';",
			wantErr:      false,
		},
		{
			name:         "Create database with engine",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engine:       new("Atomic"),
			want:         "CREATE DATABASE `database` ENGINE = Atomic;",
			wantErr:      false,
		},
		{
			name:         "Create database with cluster, engine and comment",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			clusterName:  new("default"),
			engine:       new("Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}')"),
			comment:      new("this is the comment"),
			want:         "CREATE DATABASE `database` ON CLUSTER 'default' ENGINE = Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}') COMMENT 'this is the comment';",
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
			if tt.engine != nil {
				q = q.WithEngine(*tt.engine)
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
