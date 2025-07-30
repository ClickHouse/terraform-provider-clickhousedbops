package querybuilder

import (
	"testing"
)

func Test_createrole(t *testing.T) {
	tests := []struct {
		name            string
		action          string
		resourceName    string
		clusterName     string
		settingsProfile string
		want            string
		wantErr         bool
	}{
		{
			name:         "Create role with simple name",
			resourceName: "writer",
			want:         "CREATE ROLE `writer`;",
			wantErr:      false,
		},
		{
			name:         "Create role with funky name",
			resourceName: "wr`iter",
			want:         "CREATE ROLE `wr\\`iter`;",
			wantErr:      false,
		},
		{
			name:         "Create role on cluster",
			resourceName: "foo",
			clusterName:  "cluster1",
			want:         "CREATE ROLE `foo` ON CLUSTER 'cluster1';",
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q CreateRoleQueryBuilder
			q = &createRoleQueryBuilder{
				resourceName: tt.resourceName,
			}

			if tt.clusterName != "" {
				q = q.WithCluster(&tt.clusterName)
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
