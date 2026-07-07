package querybuilder

import (
	"testing"
)

func Test_alterRoleQueryBuilder_Build(t *testing.T) {
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
			newName: new("test"),
			want:    "ALTER ROLE `foo` RENAME TO `test`;",
			wantErr: false,
		},
		{
			name:        "Change name on cluster",
			newName:     new("test"),
			clusterName: new("cluster1"),
			want:        "ALTER ROLE `foo` RENAME TO `test` ON CLUSTER 'cluster1';",
			wantErr:     false,
		},
		{
			name:               "Add profile",
			newSettingsProfile: new("profile1"),
			want:               "ALTER ROLE `foo` ADD PROFILE 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Replace profile",
			newSettingsProfile: new("profile1"),
			oldSettingsProfile: new("old"),
			want:               "ALTER ROLE `foo` DROP PROFILES 'old' ADD PROFILE 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Add profile on cluster",
			newSettingsProfile: new("profile1"),
			clusterName:        new("cluster1"),
			want:               "ALTER ROLE `foo` ON CLUSTER 'cluster1' ADD PROFILE 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Replace profile on cluster",
			newSettingsProfile: new("profile1"),
			oldSettingsProfile: new("old"),
			clusterName:        new("cluster1"),
			want:               "ALTER ROLE `foo` ON CLUSTER 'cluster1' DROP PROFILES 'old' ADD PROFILE 'profile1';",
			wantErr:            false,
		},
		{
			name:    "No profile set",
			want:    "",
			wantErr: true,
		},
		{
			name:               "Same profile set",
			newSettingsProfile: new("profile1"),
			oldSettingsProfile: new("profile1"),
			want:               "",
			wantErr:            true,
		},
		{
			name:    "Same role name set",
			newName: new("foo"),
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &alterRoleQueryBuilder{
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
