package querybuilder

import (
	"testing"
)

func Test_createSettingsProfileQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		clusterName *string
		want        string
		wantErr     bool
	}{
		{
			name:        "Simple case",
			profileName: "prf1",
			clusterName: nil,
			want:        "CREATE SETTINGS PROFILE `prf1`;",
			wantErr:     false,
		},
		{
			name:        "on cluster",
			profileName: "prf1",
			clusterName: strPtr("cluster1"),
			want:        "CREATE SETTINGS PROFILE `prf1` ON CLUSTER 'cluster1';",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &createSettingsProfileQueryBuilder{
				profileName: tt.profileName,
				clusterName: tt.clusterName,
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
