package querybuilder

import "testing"

func Test_identifiedClause(t *testing.T) {
	tests := []struct {
		name    string
		methods []AuthMethod
		want    string
	}{
		{"empty", nil, ""},
		{"no_password", []AuthMethod{{Type: IdentificationNoPassword}}, "IDENTIFIED WITH no_password"},
		{"sha256_hash", []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"h"}}}, "IDENTIFIED WITH sha256_hash BY 'h'"},
		{"sha256_hash with salt", []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"h", "s"}}}, "IDENTIFIED WITH sha256_hash BY 'h' SALT 's'"},
		{"plaintext_password", []AuthMethod{{Type: IdentificationPlaintextPassword, Args: []string{"p"}}}, "IDENTIFIED WITH plaintext_password BY 'p'"},
		{"ssl cn", []AuthMethod{{Type: IdentificationSSLCertificateCN, Args: []string{"cn"}}}, "IDENTIFIED WITH ssl_certificate CN 'cn'"},
		{"ssl san", []AuthMethod{{Type: IdentificationSSLCertificateSAN, Args: []string{"san"}}}, "IDENTIFIED WITH ssl_certificate SAN 'san'"},
		{"ssh_key", []AuthMethod{{Type: IdentificationSSHKey, Args: []string{"pub", "ssh-rsa"}}}, "IDENTIFIED WITH ssh_key BY KEY 'pub' TYPE 'ssh-rsa'"},
		{"ldap", []AuthMethod{{Type: IdentificationLDAP, Args: []string{"srv"}}}, "IDENTIFIED WITH ldap SERVER 'srv'"},
		{"kerberos no realm", []AuthMethod{{Type: IdentificationKerberos}}, "IDENTIFIED WITH kerberos"},
		{"kerberos empty realm", []AuthMethod{{Type: IdentificationKerberos, Args: []string{""}}}, "IDENTIFIED WITH kerberos"},
		{"kerberos realm", []AuthMethod{{Type: IdentificationKerberos, Args: []string{"R"}}}, "IDENTIFIED WITH kerberos REALM 'R'"},
		{"http server", []AuthMethod{{Type: IdentificationHTTPServer, Args: []string{"s"}}}, "IDENTIFIED WITH http SERVER 's'"},
		{"http scheme", []AuthMethod{{Type: IdentificationHTTPScheme, Args: []string{"Basic"}}}, "IDENTIFIED WITH http SCHEME 'Basic'"},
		{"required empty value renders faithfully", []AuthMethod{{Type: IdentificationPlaintextPassword, Args: []string{""}}}, "IDENTIFIED WITH plaintext_password BY ''"},
		{"required empty ssl san renders faithfully", []AuthMethod{{Type: IdentificationSSLCertificateSAN, Args: []string{""}}}, "IDENTIFIED WITH ssl_certificate SAN ''"},
		{"optional empty salt is omitted", []AuthMethod{{Type: IdentificationSHA256Hash, Args: []string{"h", ""}}}, "IDENTIFIED WITH sha256_hash BY 'h'"},
		{
			"multiple",
			[]AuthMethod{
				{Type: IdentificationSSLCertificateCN, Args: []string{"a"}},
				{Type: IdentificationSSLCertificateCN, Args: []string{"b"}},
				{Type: IdentificationBcryptPassword, Args: []string{"pw"}},
			},
			"IDENTIFIED WITH ssl_certificate CN 'a', ssl_certificate CN 'b', bcrypt_password BY 'pw'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := identifiedClause(tt.methods); got != tt.want {
				t.Errorf("identifiedClause() = %q, want %q", got, tt.want)
			}
		})
	}
}
