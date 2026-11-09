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
		hostIPs            *[]string
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
		{
			name:    "Set one host IP",
			hostIPs: &[]string{"10.0.0.1"},
			want:    "ALTER USER `foo` HOST IP '10.0.0.1';",
			wantErr: false,
		},
		{
			name:    "Set many host IPs",
			hostIPs: &[]string{"127.0.0.1", "::1", "10.0.0.0/8"},
			want:    "ALTER USER `foo` HOST IP '127.0.0.1', IP '::1', IP '10.0.0.0/8';",
			wantErr: false,
		},
		{
			name:        "Set host IPs on cluster",
			hostIPs:     &[]string{"10.0.0.1", "10.0.0.2"},
			clusterName: new("cluster1"),
			want:        "ALTER USER `foo` ON CLUSTER 'cluster1' HOST IP '10.0.0.1', IP '10.0.0.2';",
			wantErr:     false,
		},
		{
			name:    "Empty host IPs allow any host",
			hostIPs: &[]string{},
			want:    "ALTER USER `foo` HOST ANY;",
			wantErr: false,
		},
		{
			name:        "Nil host IPs allow any host on cluster",
			hostIPs:     new([]string(nil)),
			clusterName: new("cluster1"),
			want:        "ALTER USER `foo` ON CLUSTER 'cluster1' HOST ANY;",
			wantErr:     false,
		},
		{
			name:    "Host IP is quoted",
			hostIPs: &[]string{"10.0.0.1' HOST ANY --"},
			want:    "ALTER USER `foo` HOST IP '10.0.0.1\\' HOST ANY --';",
			wantErr: false,
		},
		{
			name:       "Change identification and host IPs",
			identified: []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"blah"}}},
			hostIPs:    &[]string{"10.0.0.1"},
			want:       "ALTER USER `foo` IDENTIFIED WITH sha256_hash BY {secret_0:String} HOST IP '10.0.0.1';",
			wantParams: map[string]string{"secret_0": "blah"},
			wantErr:    false,
		},
		{
			name:               "Change host IPs and profile",
			hostIPs:            &[]string{"10.0.0.1"},
			newSettingsProfile: new("profile1"),
			want:               "ALTER USER `foo` HOST IP '10.0.0.1' ADD PROFILES 'profile1';",
			wantErr:            false,
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
				hostIPs:            tt.hostIPs,
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
