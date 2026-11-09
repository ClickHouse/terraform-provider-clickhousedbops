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
		clusterName  *string
		identified   string
		engineName   *string
		engineArgs   []string
		settings     map[string]string
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
			engineName:   new("Replicated"),
			engineArgs:   []string{"'/clickhouse/databases/{uuid}'", "'{shard}'", "'{replica}'"},
			want:         "CREATE DATABASE `database` ENGINE = Replicated('/clickhouse/databases/{uuid}', '{shard}', '{replica}');",
		},
		{
			name:         "Create database with engine settings in deterministic order and query parameters kept verbatim",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engineName:   new("DataLakeCatalog"),
			engineArgs:   []string{"{catalog_url:String}"},
			settings: map[string]string{
				"warehouse":    "'main'",
				"catalog_type": "'rest'",
			},
			want: "CREATE DATABASE `database` ENGINE = DataLakeCatalog({catalog_url:String}) SETTINGS `catalog_type` = 'rest', `warehouse` = 'main';",
		},
		{
			name:         "Create database with all clauses",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			clusterName:  new("default"),
			engineName:   new("Atomic"),
			settings:     map[string]string{"lazy_load_tables": "1"},
			comment:      new("database comment"),
			want:         "CREATE DATABASE `database` ON CLUSTER 'default' ENGINE = Atomic SETTINGS `lazy_load_tables` = 1 COMMENT 'database comment';",
		},
		{
			name:         "Empty engine is rejected",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engineName:   new(""),
			wantErr:      true,
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
			if tt.engineName != nil {
				q = q.WithEngine(*tt.engineName, tt.engineArgs, tt.settings)
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
