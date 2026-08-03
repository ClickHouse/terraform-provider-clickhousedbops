package querybuilder

import (
	"fmt"
	"strings"
)

// Identification is the internal render-key for an auth method. It variant-encodes the
// ClickHouse methods that select a keyword for their value (ssl_certificate CN/SAN, http
// SERVER/SCHEME); every other key equals the ClickHouse auth_type.
type Identification string

const (
	IdentificationNoPassword         Identification = "no_password"
	IdentificationPlaintextPassword  Identification = "plaintext_password"
	IdentificationSHA256Password     Identification = "sha256_password"
	IdentificationSHA256Hash         Identification = "sha256_hash"
	IdentificationDoubleSHA1Password Identification = "double_sha1_password"
	IdentificationDoubleSHA1Hash     Identification = "double_sha1_hash"
	IdentificationBcryptPassword     Identification = "bcrypt_password"
	IdentificationBcryptHash         Identification = "bcrypt_hash"
	IdentificationSSLCertificateCN   Identification = "ssl_certificate_cn"
	IdentificationSSLCertificateSAN  Identification = "ssl_certificate_san"
	IdentificationHTTPServer         Identification = "http_server"
	IdentificationHTTPScheme         Identification = "http_scheme"
	IdentificationSSHKey             Identification = "ssh_key"
	IdentificationLDAP               Identification = "ldap"
	IdentificationKerberos           Identification = "kerberos"
)

// AuthMethod is a single resolved authentication method to render into an IDENTIFIED WITH clause.
// Args are positional: Args[i] is rendered with the method's i-th keyword (see methodRenderSpec).
type AuthMethod struct {
	Type Identification
	Args []string
}

// methodArg is description of a single argument for auth method.
type methodArg struct {
	// keyword is a token printed before the value.
	keyword string
	// optional means an empty value omits the whole argument.
	optional bool
	// secret means values should be rendered as parameter to avoid secret values leak.
	secret bool
}

// methodRenderSpec maps a render-key to the ClickHouse auth_type and its ordered keyword args.
type methodRenderSpec struct {
	chType string
	args   []methodArg
}

var methodRenderSpecs = map[Identification]methodRenderSpec{
	IdentificationNoPassword:         {chType: "no_password"},
	IdentificationPlaintextPassword:  {chType: "plaintext_password", args: []methodArg{{keyword: "BY", secret: true}}},
	IdentificationSHA256Password:     {chType: "sha256_password", args: []methodArg{{keyword: "BY", secret: true}}},
	IdentificationSHA256Hash:         {chType: "sha256_hash", args: []methodArg{{keyword: "BY", secret: true}, {keyword: "SALT", optional: true}}},
	IdentificationDoubleSHA1Password: {chType: "double_sha1_password", args: []methodArg{{keyword: "BY", secret: true}}},
	IdentificationDoubleSHA1Hash:     {chType: "double_sha1_hash", args: []methodArg{{keyword: "BY", secret: true}}},
	IdentificationBcryptPassword:     {chType: "bcrypt_password", args: []methodArg{{keyword: "BY", secret: true}}},
	IdentificationBcryptHash:         {chType: "bcrypt_hash", args: []methodArg{{keyword: "BY", secret: true}}},
	IdentificationSSLCertificateCN:   {chType: "ssl_certificate", args: []methodArg{{keyword: "CN"}}},
	IdentificationSSLCertificateSAN:  {chType: "ssl_certificate", args: []methodArg{{keyword: "SAN"}}},
	IdentificationHTTPServer:         {chType: "http", args: []methodArg{{keyword: "SERVER"}}},
	IdentificationHTTPScheme:         {chType: "http", args: []methodArg{{keyword: "SCHEME"}}},
	IdentificationSSHKey:             {chType: "ssh_key", args: []methodArg{{keyword: "BY KEY"}, {keyword: "TYPE"}}},
	IdentificationLDAP:               {chType: "ldap", args: []methodArg{{keyword: "SERVER"}}},
	IdentificationKerberos:           {chType: "kerberos", args: []methodArg{{keyword: "REALM", optional: true}}},
}

// identifiedClause renders "IDENTIFIED WITH m1, m2, ..." for the given methods and the server-side
// query parameters carrying any secret values
func identifiedClause(methods []AuthMethod) (string, map[string]string) {
	if len(methods) == 0 {
		return "", nil
	}

	params := map[string]string{}
	clauses := make([]string, 0, len(methods))
	for _, m := range methods {
		spec := methodRenderSpecs[m.Type]
		parts := []string{spec.chType}
		for i, a := range spec.args {
			if i >= len(m.Args) {
				break
			}
			if a.optional && m.Args[i] == "" {
				continue
			}
			if a.secret {
				name := fmt.Sprintf("secret_%d", len(params))
				parts = append(parts, a.keyword, fmt.Sprintf("{%s:String}", name))
				params[name] = m.Args[i]
			} else {
				parts = append(parts, a.keyword, quote(m.Args[i]))
			}
		}
		clauses = append(clauses, strings.Join(parts, " "))
	}

	return "IDENTIFIED WITH " + strings.Join(clauses, ", "), params
}
