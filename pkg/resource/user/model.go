package user

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type User struct {
	ClusterName                 types.String `tfsdk:"cluster_name"`
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	PasswordSha256Hash          types.String `tfsdk:"password_sha256_hash"`
	PasswordSha256HashWO        types.String `tfsdk:"password_sha256_hash_wo"`
	PasswordSha256HashVersionWO types.Int32  `tfsdk:"password_sha256_hash_wo_version"`
	HostIPs                     types.Set    `tfsdk:"host_ips"`
	Auth                        *AuthModel   `tfsdk:"auth"`
}

type AuthModel struct {
	NoPassword         *NoPasswordModel      `tfsdk:"no_password"`
	PlaintextPassword  []SecretMethodModel   `tfsdk:"plaintext_password"`
	Sha256Password     []SecretMethodModel   `tfsdk:"sha256_password"`
	Sha256Hash         []Sha256HashModel     `tfsdk:"sha256_hash"`
	DoubleSha1Password []SecretMethodModel   `tfsdk:"double_sha1_password"`
	DoubleSha1Hash     []SecretMethodModel   `tfsdk:"double_sha1_hash"`
	BcryptPassword     []SecretMethodModel   `tfsdk:"bcrypt_password"`
	BcryptHash         []SecretMethodModel   `tfsdk:"bcrypt_hash"`
	SSLCertificate     []SSLCertificateModel `tfsdk:"ssl_certificate"`
	HTTP               []HTTPModel           `tfsdk:"http"`
	SSHKey             []SSHKeyModel         `tfsdk:"ssh_key"`
	LDAP               []LDAPModel           `tfsdk:"ldap"`
	Kerberos           []KerberosModel       `tfsdk:"kerberos"`
}

// NoPasswordModel is an empty presence block: set when passwordless auth is desired.
type NoPasswordModel struct{}

type SecretMethodModel struct {
	Value          types.String `tfsdk:"value"`
	ValueWO        types.String `tfsdk:"value_wo"`
	ValueWOVersion types.Int32  `tfsdk:"value_wo_version"`
}

type Sha256HashModel struct {
	SecretMethodModel
	Salt types.String `tfsdk:"salt"`
}

type SSLCertificateModel struct {
	CommonName     types.String `tfsdk:"common_name"`
	SubjectAltName types.String `tfsdk:"subject_alt_name"`
}

type HTTPModel struct {
	Server types.String `tfsdk:"server"`
	Scheme types.String `tfsdk:"scheme"`
}

type SSHKeyModel struct {
	PublicKey types.String `tfsdk:"public_key"`
	Type      types.String `tfsdk:"type"`
}

type LDAPModel struct {
	Server types.String `tfsdk:"server"`
}

type KerberosModel struct {
	Realm types.String `tfsdk:"realm"`
}
