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
		engineName   *string
		engineArgs   []string
		settings     map[string]string
		parameters   map[string]string
		want         string
		wantRedacted string
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
			want:         "CREATE DATABASE `database` ENGINE = `Replicated`('/clickhouse/databases/{uuid}', '{shard}', '{replica}');",
		},
		{
			name:         "Create database with engine settings in deterministic order",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engineName:   new("DataLakeCatalog"),
			engineArgs:   []string{"'https://catalog.example.test'"},
			settings: map[string]string{ //nolint:gosec // Contains a placeholder, not a credential.
				"warehouse":          "'main'",
				"catalog_credential": "{credential:String}",
				"catalog_type":       "'rest'",
			},
			parameters:   map[string]string{"credential": "secret"},
			want:         "CREATE DATABASE `database` ENGINE = `DataLakeCatalog`('https://catalog.example.test') SETTINGS `catalog_credential` = 'secret', `catalog_type` = 'rest', `warehouse` = 'main';",
			wantRedacted: "CREATE DATABASE `database` ENGINE = `DataLakeCatalog`('https://catalog.example.test') SETTINGS `catalog_credential` = '[REDACTED]', `catalog_type` = 'rest', `warehouse` = 'main';",
		},
		{
			name:         "Create database with all clauses",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			clusterName:  new("default"),
			engineName:   new("Atomic"),
			settings:     map[string]string{"lazy_load_tables": "1"},
			comment:      new("database comment"),
			want:         "CREATE DATABASE `database` ON CLUSTER 'default' ENGINE = `Atomic` SETTINGS `lazy_load_tables` = 1 COMMENT 'database comment';",
		},
		{
			name:         "Empty engine is rejected",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engineName:   new(""),
			wantErr:      true,
		},
		{
			name:         "Unconfigured parameter is rejected",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engineName:   new("DataLakeCatalog"),
			settings: map[string]string{ //nolint:gosec // Contains a placeholder, not a credential.
				"catalog_credential": "{credential:String}",
			},
			wantErr: true,
		},
		{
			name:         "Unused parameter is rejected",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engineName:   new("DataLakeCatalog"),
			parameters:   map[string]string{"credential": "secret"},
			wantErr:      true,
		},
		{
			name:         "Parameter values are substituted in one pass",
			resourceType: resourceTypeDatabase,
			resourceName: "database",
			engineName:   new("DataLakeCatalog"),
			settings: map[string]string{ //nolint:gosec // Contains a placeholder, not a credential.
				"catalog_credential": "{credential:String}",
				"auth_scope":         "{scope:String}",
			},
			parameters: map[string]string{ //nolint:gosec // Test values are placeholders, not credentials.
				"credential": "{scope:String}",
				"scope":      "all-apis",
			},
			want:         "CREATE DATABASE `database` ENGINE = `DataLakeCatalog` SETTINGS `auth_scope` = 'all-apis', `catalog_credential` = '{scope:String}';",
			wantRedacted: "CREATE DATABASE `database` ENGINE = `DataLakeCatalog` SETTINGS `auth_scope` = '[REDACTED]', `catalog_credential` = '[REDACTED]';",
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
			if tt.parameters != nil {
				q = q.WithParameters(tt.parameters)
			}

			got, err := q.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Build() got = %v, want %v", got, tt.want)
			}
			if tt.wantErr {
				return
			}
			redacted, err := q.RedactedQuery()
			if err != nil {
				t.Errorf("RedactedQuery() error = %v", err)
			}
			wantRedacted := tt.wantRedacted
			if wantRedacted == "" {
				wantRedacted = tt.want
			}
			if redacted != wantRedacted {
				t.Errorf("RedactedQuery() got = %v, want %v", redacted, wantRedacted)
			}
		})
	}
}
