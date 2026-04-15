package querybuilder

import (
	"testing"
)

func Test_createuser(t *testing.T) {
	tests := []struct {
		name            string
		resourceName    string
		identifiedWith  Identification
		identifiedBy    string
		hostIPs         []string
		settingsProfile string
		want            string
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
			name:           "Create user with simple name and password",
			resourceName:   "john",
			identifiedWith: IdentificationSHA256Hash,
			identifiedBy:   "blah",
			want:           "CREATE USER `john` IDENTIFIED WITH sha256_hash BY 'blah';",
			wantErr:        false,
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
			name:           "Create user with host IP and password",
			resourceName:   "mira",
			hostIPs:        []string{"127.0.0.1"},
			identifiedWith: IdentificationSHA256Hash,
			identifiedBy:   "blah",
			want:           "CREATE USER `mira` HOST IP '127.0.0.1' IDENTIFIED WITH sha256_hash BY 'blah';",
			wantErr:        false,
		},
		{
			name:         "Create user with multiple host IP restrictions",
			resourceName: "mira",
			hostIPs:      []string{"127.0.0.1", "192.168.1.1", "10.0.0.1"},
			want:         "CREATE USER `mira` HOST IP '127.0.0.1' HOST IP '192.168.1.1' HOST IP '10.0.0.1';",
			wantErr:      false,
		},
		{
			name:           "Create user with ssl_certificate auth",
			resourceName:   "teleport_cert_read",
			identifiedWith: IdentificationSSLCertificate,
			identifiedBy:   "teleport_cert_read",
			want:           "CREATE USER `teleport_cert_read` IDENTIFIED WITH ssl_certificate CN 'teleport_cert_read';",
			wantErr:        false,
		},
		{
			name:           "Create user with ssl_certificate and host IP",
			resourceName:   "cert_user",
			identifiedWith: IdentificationSSLCertificate,
			identifiedBy:   "cert_user_cn",
			hostIPs:        []string{"10.0.0.1"},
			want:           "CREATE USER `cert_user` HOST IP '10.0.0.1' IDENTIFIED WITH ssl_certificate CN 'cert_user_cn';",
			wantErr:        false,
		},
		{
			name:           "Create user with plaintext_password auth",
			resourceName:   "plain_user",
			identifiedWith: IdentificationPlaintextPassword,
			identifiedBy:   "mypassword",
			want:           "CREATE USER `plain_user` IDENTIFIED WITH plaintext_password BY 'mypassword';",
			wantErr:        false,
		},
		{
			name:           "Create user with bcrypt_hash auth",
			resourceName:   "bcrypt_user",
			identifiedWith: IdentificationBcryptHash,
			identifiedBy:   "$2a$10$abc123",
			want:           "CREATE USER `bcrypt_user` IDENTIFIED WITH bcrypt_hash BY '$2a$10$abc123';",
			wantErr:        false,
		},
		{
			name:           "Create user with double_sha1_hash auth",
			resourceName:   "sha1_user",
			identifiedWith: IdentificationDoubleSHA1Hash,
			identifiedBy:   "abcdef1234567890",
			want:           "CREATE USER `sha1_user` IDENTIFIED WITH double_sha1_hash BY 'abcdef1234567890';",
			wantErr:        false,
		},
		{
			name:           "Create user with no_password auth",
			resourceName:   "nopass_user",
			identifiedWith: IdentificationNoPassword,
			identifiedBy:   "",
			want:           "CREATE USER `nopass_user` IDENTIFIED WITH no_password;",
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var q CreateUserQueryBuilder
			q = &createUserQueryBuilder{
				resourceName: tt.resourceName,
			}

			if len(tt.hostIPs) > 0 {
				q = q.HostIPs(tt.hostIPs)
			}

			if tt.identifiedWith != "" {
				q = q.Identified(tt.identifiedWith, tt.identifiedBy)
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
		})
	}
}
