package querybuilder

import (
	"testing"
)

func Test_selectNamedCollectionKeysQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		clusterName    *string
		want           string
		wantErr        bool
	}{
		{
			name:           "Simple case",
			collectionName: "collection1",
			want:           "SELECT kv.1 AS key_name, kv.2 AS key_value FROM `system`.`named_collections` ARRAY JOIN `collection` AS kv WHERE `name` = 'collection1';",
			wantErr:        false,
		},
		{
			name:           "On cluster",
			collectionName: "collection1",
			clusterName:    strPtr("cluster1"),
			want:           "SELECT kv.1 AS key_name, kv.2 AS key_value FROM cluster('cluster1', `system`.`named_collections`) ARRAY JOIN `collection` AS kv WHERE `name` = 'collection1';",
			wantErr:        false,
		},
		{
			name:           "Escaping",
			collectionName: "col'lection",
			want:           "SELECT kv.1 AS key_name, kv.2 AS key_value FROM `system`.`named_collections` ARRAY JOIN `collection` AS kv WHERE `name` = 'col\\'lection';",
			wantErr:        false,
		},
		{
			name:           "Fail with empty collection name",
			collectionName: "",
			want:           "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &selectNamedCollectionKeysQueryBuilder{
				collectionName: tt.collectionName,
				clusterName:    tt.clusterName,
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
