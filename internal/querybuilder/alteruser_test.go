package querybuilder

import (
	"testing"
)

func Test_alterUserQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name               string
		oldSettingsProfile *string
		newSettingsProfile *string
		newName            *string
		clusterName        *string
		want               string
		wantErr            bool
	}{
		{
			name:    "Change name",
			newName: strPtr("test"),
			want:    "ALTER USER `foo` RENAME TO `test`;",
			wantErr: false,
		},
		{
			name:        "Change name on cluster",
			newName:     strPtr("test"),
			clusterName: strPtr("cluster1"),
			want:        "ALTER USER `foo` RENAME TO `test` ON CLUSTER 'cluster1';",
			wantErr:     false,
		},
		{
			name:               "Add profile",
			newSettingsProfile: strPtr("profile1"),
			want:               "ALTER USER `foo` ADD PROFILES 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Replace profile",
			newSettingsProfile: strPtr("profile1"),
			oldSettingsProfile: strPtr("old"),
			want:               "ALTER USER `foo` DROP PROFILES 'old' ADD PROFILES 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Add profile on cluster",
			newSettingsProfile: strPtr("profile1"),
			clusterName:        strPtr("cluster1"),
			want:               "ALTER USER `foo` ON CLUSTER 'cluster1' ADD PROFILES 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Replace profile on cluster",
			newSettingsProfile: strPtr("profile1"),
			oldSettingsProfile: strPtr("old"),
			clusterName:        strPtr("cluster1"),
			want:               "ALTER USER `foo` ON CLUSTER 'cluster1' DROP PROFILES 'old' ADD PROFILES 'profile1';",
			wantErr:            false,
		},
		{
			name:    "No profile set",
			want:    "",
			wantErr: true,
		},
		{
			name:               "Same profile set",
			newSettingsProfile: strPtr("profile1"),
			oldSettingsProfile: strPtr("profile1"),
			want:               "",
			wantErr:            true,
		},
		{
			name:    "Same username set",
			newName: strPtr("foo"),
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &alterUserQueryBuilder{
				resourceName:       "foo",
				oldSettingsProfile: tt.oldSettingsProfile,
				newSettingsProfile: tt.newSettingsProfile,
				newName:            tt.newName,
				clusterName:        tt.clusterName,
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
