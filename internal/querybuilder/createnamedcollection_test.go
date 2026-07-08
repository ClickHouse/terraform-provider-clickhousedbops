package querybuilder

import (
	"testing"
)

func Test_createNamedCollectionQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		clusterName    *string
		keys           []namedCollectionKeyData
		want           string
		wantErr        bool
	}{
		{
			name:           "Single key",
			collectionName: "collection1",
			keys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/"},
			},
			want:    "CREATE NAMED COLLECTION `collection1` AS `url` = 'https://example.com/';",
			wantErr: false,
		},
		{
			name:           "Multiple keys with flags",
			collectionName: "collection1",
			keys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/", Overridable: boolPtr(true)},
				{Name: "secret_access_key", Value: "topsecret", Overridable: boolPtr(false)},
				{Name: "format", Value: "CSV"},
			},
			want:    "CREATE NAMED COLLECTION `collection1` AS `url` = 'https://example.com/' OVERRIDABLE, `secret_access_key` = 'topsecret' NOT OVERRIDABLE, `format` = 'CSV';",
			wantErr: false,
		},
		{
			name:           "On cluster",
			collectionName: "collection1",
			clusterName:    strPtr("cluster1"),
			keys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/"},
			},
			want:    "CREATE NAMED COLLECTION `collection1` ON CLUSTER 'cluster1' AS `url` = 'https://example.com/';",
			wantErr: false,
		},
		{
			name:           "Escaping",
			collectionName: "col`lection",
			keys: []namedCollectionKeyData{
				{Name: "we`ird", Value: "it's"},
			},
			want:    "CREATE NAMED COLLECTION `col\\`lection` AS `we\\`ird` = 'it\\'s';",
			wantErr: false,
		},
		{
			name:           "Fail with empty collection name",
			collectionName: "",
			keys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/"},
			},
			want:    "",
			wantErr: true,
		},
		{
			name:           "Fail with no keys",
			collectionName: "collection1",
			keys:           []namedCollectionKeyData{},
			want:           "",
			wantErr:        true,
		},
		{
			name:           "Fail with empty key name",
			collectionName: "collection1",
			keys: []namedCollectionKeyData{
				{Name: "", Value: "value"},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &createNamedCollectionQueryBuilder{
				collectionName: tt.collectionName,
				clusterName:    tt.clusterName,
				keys:           tt.keys,
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

func boolPtr(val bool) *bool {
	return &val
}
