package querybuilder

import (
	"maps"
	"testing"
)

func Test_alterUserQueryBuilder_Build(t *testing.T) {
	tests := []struct {
		name               string
		identified         []AuthMethod
		oldSettingsProfile *string
		newSettingsProfile *string
		newName            *string
		clusterName        *string
		want               string
		wantParams         map[string]string
		wantErr            bool
	}{
		{
			name:    "Change name",
			newName: new("test"),
			want:    "ALTER USER `foo` RENAME TO `test`;",
			wantErr: false,
		},
		{
			name:        "Change name on cluster",
			newName:     new("test"),
			clusterName: new("cluster1"),
			want:        "ALTER USER `foo` RENAME TO `test` ON CLUSTER 'cluster1';",
			wantErr:     false,
		},
		{
			name:               "Add profile",
			newSettingsProfile: new("profile1"),
			want:               "ALTER USER `foo` ADD PROFILES 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Replace profile",
			newSettingsProfile: new("profile1"),
			oldSettingsProfile: new("old"),
			want:               "ALTER USER `foo` DROP PROFILES 'old' ADD PROFILES 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Add profile on cluster",
			newSettingsProfile: new("profile1"),
			clusterName:        new("cluster1"),
			want:               "ALTER USER `foo` ON CLUSTER 'cluster1' ADD PROFILES 'profile1';",
			wantErr:            false,
		},
		{
			name:               "Replace profile on cluster",
			newSettingsProfile: new("profile1"),
			oldSettingsProfile: new("old"),
			clusterName:        new("cluster1"),
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
			newSettingsProfile: new("profile1"),
			oldSettingsProfile: new("profile1"),
			want:               "",
			wantErr:            true,
		},
		{
			name:    "Same username set",
			newName: new("foo"),
			want:    "",
			wantErr: true,
		},
		{
			name:       "Change identification",
			identified: []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"blah"}}},
			want:       "ALTER USER `foo` IDENTIFIED WITH sha256_hash BY {secret_0:String};",
			wantParams: map[string]string{"secret_0": "blah"},
			wantErr:    false,
		},
		{
			name:        "Change identification on cluster",
			identified:  []AuthMethod{{Type: IdentificationSSLCertificateCN, Args: []string{"cn"}}},
			clusterName: new("cluster1"),
			want:        "ALTER USER `foo` ON CLUSTER 'cluster1' IDENTIFIED WITH ssl_certificate CN 'cn';",
			wantErr:     false,
		},
		{
			name:       "Rename and change identification",
			newName:    new("test"),
			identified: []AuthMethod{{Type: IdentificationNoPassword}},
			want:       "ALTER USER `foo` RENAME TO `test` IDENTIFIED WITH no_password;",
			wantErr:    false,
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
			q.Identified(tt.identified)
			got, err := q.Build()
			if (err != nil) != tt.wantErr {
				t.Errorf("Build() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Build() got = %v, want %v", got, tt.want)
			}
			if !maps.Equal(q.Parameters(), tt.wantParams) {
				t.Errorf("Parameters() = %v, want %v", q.Parameters(), tt.wantParams)
			}
		})
	}
}
