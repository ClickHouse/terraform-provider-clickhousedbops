package querybuilder

import (
	"testing"
)

func Test_alterNamedCollectionQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name           string
		collectionName string
		clusterName    *string
		setKeys        []namedCollectionKeyData
		deleteKeys     []string
		want           string
		wantErr        bool
	}{
		{
			name:           "Set single key",
			collectionName: "collection1",
			setKeys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/"},
			},
			want:    "ALTER NAMED COLLECTION `collection1` SET `url` = 'https://example.com/';",
			wantErr: false,
		},
		{
			name:           "Set keys with flags",
			collectionName: "collection1",
			setKeys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/", Overridable: boolPtr(true)},
				{Name: "secret", Value: "topsecret", Overridable: boolPtr(false)},
			},
			want:    "ALTER NAMED COLLECTION `collection1` SET `url` = 'https://example.com/' OVERRIDABLE, `secret` = 'topsecret' NOT OVERRIDABLE;",
			wantErr: false,
		},
		{
			name:           "Delete keys",
			collectionName: "collection1",
			deleteKeys:     []string{"old1", "old2"},
			want:           "ALTER NAMED COLLECTION `collection1` DELETE `old1`, `old2`;",
			wantErr:        false,
		},
		{
			name:           "Set and delete combined",
			collectionName: "collection1",
			setKeys: []namedCollectionKeyData{
				{Name: "url", Value: "https://new.example.com/"},
			},
			deleteKeys: []string{"old1"},
			want:       "ALTER NAMED COLLECTION `collection1` SET `url` = 'https://new.example.com/', DELETE `old1`;",
			wantErr:    false,
		},
		{
			name:           "On cluster",
			collectionName: "collection1",
			clusterName:    strPtr("cluster1"),
			setKeys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/"},
			},
			want:    "ALTER NAMED COLLECTION `collection1` ON CLUSTER 'cluster1' SET `url` = 'https://example.com/';",
			wantErr: false,
		},
		{
			name:           "Fail with empty collection name",
			collectionName: "",
			setKeys: []namedCollectionKeyData{
				{Name: "url", Value: "https://example.com/"},
			},
			want:    "",
			wantErr: true,
		},
		{
			name:           "Fail with no keys",
			collectionName: "collection1",
			want:           "",
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &alterNamedCollectionQueryBuilder{
				collectionName: tt.collectionName,
				clusterName:    tt.clusterName,
				setKeys:        tt.setKeys,
				deleteKeys:     tt.deleteKeys,
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
