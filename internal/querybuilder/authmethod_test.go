package querybuilder

import (
	"maps"
	"testing"
)

func Test_identifiedClause(t *testing.T) {
	tests := []struct {
		name       string
		methods    []AuthMethod
		want       string
		wantParams map[string]string
	}{
		{"empty", nil, "", nil},
		{"no_password", []AuthMethod{{Type: IdentificationNoPassword}}, "IDENTIFIED WITH no_password", nil},
		{"sha256_hash", []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"h"}}}, "IDENTIFIED WITH sha256_hash BY {secret_0:String}", map[string]string{"secret_0": "h"}},
		{"sha256_hash with salt", []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"h", "s"}}}, "IDENTIFIED WITH sha256_hash BY {secret_0:String} SALT 's'", map[string]string{"secret_0": "h"}},
		{"plaintext_password", []AuthMethod{{Type: IdentificationPlaintextPassword, Args: []string{"p"}}}, "IDENTIFIED WITH plaintext_password BY {secret_0:String}", map[string]string{"secret_0": "p"}},
		{"ssl cn", []AuthMethod{{Type: IdentificationSSLCertificateCN, Args: []string{"cn"}}}, "IDENTIFIED WITH ssl_certificate CN 'cn'", nil},
		{"ssl san", []AuthMethod{{Type: IdentificationSSLCertificateSAN, Args: []string{"san"}}}, "IDENTIFIED WITH ssl_certificate SAN 'san'", nil},
		{"ssh_key", []AuthMethod{{Type: IdentificationSSHKey, Args: []string{"pub", "ssh-rsa"}}}, "IDENTIFIED WITH ssh_key BY KEY 'pub' TYPE 'ssh-rsa'", nil},
		{"ldap", []AuthMethod{{Type: IdentificationLDAP, Args: []string{"srv"}}}, "IDENTIFIED WITH ldap SERVER 'srv'", nil},
		{"kerberos no realm", []AuthMethod{{Type: IdentificationKerberos}}, "IDENTIFIED WITH kerberos", nil},
		{"kerberos empty realm", []AuthMethod{{Type: IdentificationKerberos, Args: []string{""}}}, "IDENTIFIED WITH kerberos", nil},
		{"kerberos realm", []AuthMethod{{Type: IdentificationKerberos, Args: []string{"R"}}}, "IDENTIFIED WITH kerberos REALM 'R'", nil},
		{"http server", []AuthMethod{{Type: IdentificationHTTPServer, Args: []string{"s"}}}, "IDENTIFIED WITH http SERVER 's'", nil},
		{"http scheme", []AuthMethod{{Type: IdentificationHTTPScheme, Args: []string{"Basic"}}}, "IDENTIFIED WITH http SCHEME 'Basic'", nil},
		{"required empty secret is parameterized", []AuthMethod{{Type: IdentificationPlaintextPassword, Args: []string{""}}}, "IDENTIFIED WITH plaintext_password BY {secret_0:String}", map[string]string{"secret_0": ""}},
		{"required empty ssl san renders faithfully", []AuthMethod{{Type: IdentificationSSLCertificateSAN, Args: []string{""}}}, "IDENTIFIED WITH ssl_certificate SAN ''", nil},
		{"optional empty salt is omitted", []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"h", ""}}}, "IDENTIFIED WITH sha256_hash BY {secret_0:String}", map[string]string{"secret_0": "h"}},
		{
			"multiple secrets get distinct parameters",
			[]AuthMethod{
				{Type: IdentificationSHA256Hash, Args: []string{"h1"}},
				{Type: IdentificationBcryptPassword, Args: []string{"pw"}},
			},
			"IDENTIFIED WITH sha256_hash BY {secret_0:String}, bcrypt_password BY {secret_1:String}",
			map[string]string{"secret_0": "h1", "secret_1": "pw"},
		},
		{
			"secret and non-secret methods combined",
			[]AuthMethod{
				{Type: IdentificationSSLCertificateCN, Args: []string{"a"}},
				{Type: IdentificationSSLCertificateCN, Args: []string{"b"}},
				{Type: IdentificationBcryptPassword, Args: []string{"pw"}},
			},
			"IDENTIFIED WITH ssl_certificate CN 'a', ssl_certificate CN 'b', bcrypt_password BY {secret_0:String}",
			map[string]string{"secret_0": "pw"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, params := identifiedClause(tt.methods)
			if got != tt.want {
				t.Errorf("identifiedClause() clause = %q, want %q", got, tt.want)
			}
			if !maps.Equal(params, tt.wantParams) {
				t.Errorf("identifiedClause() params = %v, want %v", params, tt.wantParams)
			}
		})
	}
}
