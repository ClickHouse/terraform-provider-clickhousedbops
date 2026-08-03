package querybuilder

import (
	"maps"
	"testing"
)

func Test_createuser(t *testing.T) {
	tests := []struct {
		name            string
		resourceName    string
		methods         []AuthMethod
		hostIPs         []string
		settingsProfile string
		want            string
		wantParams      map[string]string
		wantErr         bool
	}{
		{
			name:         "Create user with simple name and no password",
			resourceName: "john",
			want:         "CREATE USER `john`;",
			wantErr:      false,
		},
		{
			name:         "Create user with funky name and no password",
			resourceName: "jo`hn",
			want:         "CREATE USER `jo\\`hn`;",
			wantErr:      false,
		},
		{
			name:         "Create user with simple name and password",
			resourceName: "john",
			methods:      []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"blah"}}},
			want:         "CREATE USER `john` IDENTIFIED WITH sha256_hash BY {secret_0:String};",
			wantParams:   map[string]string{"secret_0": "blah"},
			wantErr:      false,
		},
		{
			name:         "Create user fails when no user name is set",
			resourceName: "",
			want:         "",
			wantErr:      true,
		},
		{
			name:            "Create user with settings profile",
			resourceName:    "foo",
			settingsProfile: "test",
			want:            "CREATE USER `foo` SETTINGS PROFILE 'test';",
			wantErr:         false,
		},
		{
			name:         "Create user with host IP restriction",
			resourceName: "mira",
			hostIPs:      []string{"127.0.0.1"},
			want:         "CREATE USER `mira` HOST IP '127.0.0.1';",
			wantErr:      false,
		},
		{
			name:         "Create user with host IP and password",
			resourceName: "mira",
			hostIPs:      []string{"127.0.0.1"},
			methods:      []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"blah"}}},
			want:         "CREATE USER `mira` HOST IP '127.0.0.1' IDENTIFIED WITH sha256_hash BY {secret_0:String};",
			wantParams:   map[string]string{"secret_0": "blah"},
			wantErr:      false,
		},
		{
			name:         "Create user with multiple host IP restrictions",
			resourceName: "mira",
			hostIPs:      []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"},
			want:         "CREATE USER `mira` HOST IP '127.0.0.1' HOST IP '192.168.1.1' HOST IP '10.0.0.1';",
			wantErr:      false,
		},
		{
			name:         "Create user with multiple auth methods",
			resourceName: "svc",
			methods: []AuthMethod{
				{Type: IdentificationSSLCertificateCN, Args: []string{"a"}},
				{Type: IdentificationBcryptPassword, Args: []string{"pw"}},
			},
			want:       "CREATE USER `svc` IDENTIFIED WITH ssl_certificate CN 'a', bcrypt_password BY {secret_0:String};",
			wantParams: map[string]string{"secret_0": "pw"},
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q CreateUserQueryBuilder = &createUserQueryBuilder{
				resourceName: tt.resourceName,
			}

			if len(tt.hostIPs) > 0 {
				q = q.HostIPs(tt.hostIPs)
			}

			if len(tt.methods) > 0 {
				q = q.Identified(tt.methods)
			}

			if tt.settingsProfile != "" {
				q = q.WithSettingsProfile(&tt.settingsProfile)
			}

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
