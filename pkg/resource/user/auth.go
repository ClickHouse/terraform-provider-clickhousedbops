package user

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/dbops"
	"github.com/ClickHouse/terraform-provider-clickhousedbops/internal/querybuilder"
)

// authMethodBlockNames are the repeatable per-type auth method blocks under `auth`.
var authMethodBlockNames = []string{
	"plaintext_password", "sha256_password", "sha256_hash",
	"double_sha1_password", "double_sha1_hash", "bcrypt_password", "bcrypt_hash",
	"ssl_certificate", "http", "ssh_key", "ldap", "kerberos",
}

func authMethodBlockPaths() []path.Expression {
	exprs := make([]path.Expression, 0, len(authMethodBlockNames))
	for _, n := range authMethodBlockNames {
		exprs = append(exprs, path.MatchRoot("auth").AtName(n))
	}
	return exprs
}

// noPasswordConflictPaths is everything no_password cannot co-exist with.
func noPasswordConflictPaths() []path.Expression {
	exprs := authMethodBlockPaths()
	return append(exprs, path.MatchRoot("password_sha256_hash"), path.MatchRoot("password_sha256_hash_wo"))
}

// authConfigValidators enforces that at least one authentication method is configured.
func authConfigValidators() []resource.ConfigValidator {
	exprs := append([]path.Expression{path.MatchRoot("auth").AtName("no_password")}, authMethodBlockPaths()...)
	exprs = append(exprs, path.MatchRoot("password_sha256_hash"), path.MatchRoot("password_sha256_hash_wo"))
	return []resource.ConfigValidator{
		resourcevalidator.AtLeastOneOf(exprs...),
	}
}

func userAuthBlock() schema.SingleNestedBlock {
	sha256Hash := secretAuthBlock("sha256_hash", "SHA256 hash authentication.",
		stringvalidator.RegexMatches(regexp.MustCompile(`^[a-fA-F0-9]{64}$`), "sha256_hash value must be a valid SHA256 hash"))
	sha256Hash.NestedObject.Attributes["salt"] = schema.StringAttribute{
		Optional:    true,
		Description: "Optional salt used with the sha256 hash.",
	}

	return schema.SingleNestedBlock{
		Description: "Authentication methods for the user. Methods may be combined and each block (except no_password) may be repeated.",
		Blocks: map[string]schema.Block{
			"no_password": schema.SingleNestedBlock{
				Description: "Passwordless authentication. Cannot be combined with any other method.",
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(noPasswordConflictPaths()...),
				},
			},
			"plaintext_password":   secretAuthBlock("plaintext_password", "Plaintext password authentication."),
			"sha256_password":      secretAuthBlock("sha256_password", "SHA256 password authentication (ClickHouse computes the hash)."),
			"sha256_hash":          sha256Hash,
			"double_sha1_password": secretAuthBlock("double_sha1_password", "Double SHA1 password authentication."),
			"double_sha1_hash":     secretAuthBlock("double_sha1_hash", "Double SHA1 hash authentication.", stringvalidator.RegexMatches(regexp.MustCompile(`^[a-fA-F0-9]{40}$`), "double_sha1_hash value must be a valid SHA1 hash")),
			"bcrypt_password":      secretAuthBlock("bcrypt_password", "Bcrypt password authentication."),
			"bcrypt_hash":          secretAuthBlock("bcrypt_hash", "Bcrypt hash authentication.", stringvalidator.RegexMatches(regexp.MustCompile(`^\$2[abxy]\$[0-9]{2}\$[A-Za-z0-9./]{53}$`), "bcrypt_hash value must be a valid bcrypt hash")),
			"ssl_certificate": schema.ListNestedBlock{
				Description: "SSL certificate authentication. Exactly one of common_name or subject_alt_name.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"common_name": schema.StringAttribute{
							Optional:    true,
							Description: "Certificate Common Name (CN).",
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("common_name"),
									path.MatchRelative().AtParent().AtName("subject_alt_name"),
								),
							},
						},
						"subject_alt_name": schema.StringAttribute{
							Optional:    true,
							Description: "Certificate Subject Alternative Name (SAN).",
						},
					},
				},
			},
			"http": schema.ListNestedBlock{
				Description: "HTTP authentication. Exactly one of server or scheme.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"server": schema.StringAttribute{
							Optional:    true,
							Description: "HTTP authentication server name.",
							Validators: []validator.String{
								stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("server"),
									path.MatchRelative().AtParent().AtName("scheme"),
								),
							},
						},
						"scheme": schema.StringAttribute{
							Optional:    true,
							Description: "HTTP authentication scheme (e.g. Basic).",
						},
					},
				},
			},
			"ssh_key": schema.ListNestedBlock{
				Description: "SSH key authentication.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"public_key": schema.StringAttribute{Required: true, Description: "SSH public key."},
						"type":       schema.StringAttribute{Required: true, Description: "SSH key type (e.g. ssh-rsa, ssh-ed25519)."},
					},
				},
			},
			"ldap": schema.ListNestedBlock{
				Description: "LDAP authentication.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"server": schema.StringAttribute{Required: true, Description: "LDAP server name (as defined in ClickHouse config)."},
					},
				},
			},
			"kerberos": schema.ListNestedBlock{
				Description: "Kerberos authentication.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"realm": schema.StringAttribute{Optional: true, Description: "Optional Kerberos realm to restrict authentication to."},
					},
				},
			},
		},
	}
}

// secretAuthAttributes is the shared shape for password/hash methods: a value or its write-only variant gated by a version.
// blockName is the method's block name under `auth`, needed for the absolute write-only path (PreferWriteOnlyAttribute matches from the config root, so a relative sibling path can't be used inside a repeatable block).
func secretAuthAttributes(blockName string, valueValidators []validator.String) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"value": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			Description: "Authentication value stored in state. Use for Terraform/OpenTofu < 1.11. Exactly one of value or value_wo.",
			Validators: append([]validator.String{
				stringvalidator.PreferWriteOnlyAttribute(path.MatchRoot("auth").AtName(blockName).AtAnyListIndex().AtName("value_wo")),
				stringvalidator.ExactlyOneOf(
					path.MatchRelative().AtParent().AtName("value"),
					path.MatchRelative().AtParent().AtName("value_wo"),
				),
			}, valueValidators...),
		},
		"value_wo": schema.StringAttribute{
			Optional:    true,
			Sensitive:   true,
			WriteOnly:   true,
			Description: "Write-only authentication value, not stored in state. Use for Terraform/OpenTofu >= 1.11.",
			Validators: append([]validator.String{
				stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("value_wo_version")),
			}, valueValidators...),
		},
		"value_wo_version": schema.Int32Attribute{
			Optional:    true,
			Description: "Version of value_wo. Bump to re-apply the write-only value.",
			Validators: []validator.Int32{
				int32validator.AlsoRequires(path.MatchRelative().AtParent().AtName("value_wo")),
			},
		},
	}

	return attrs
}

func secretAuthBlock(blockName string, description string, valueValidators ...validator.String) schema.ListNestedBlock {
	return schema.ListNestedBlock{
		Description:  description,
		NestedObject: schema.NestedBlockObject{Attributes: secretAuthAttributes(blockName, valueValidators)},
	}
}

// resolveAuthMethods flattens the configured legacy password fields and the `auth` block into the
// full ordered set of methods to assert. Write-only values are read from config.
func resolveAuthMethods(plan, config User) []dbops.AuthMethod {
	var methods []dbops.AuthMethod

	switch {
	case !plan.PasswordSha256Hash.IsNull():
		methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationSHA256Hash), Args: []string{plan.PasswordSha256Hash.ValueString()}})
	case !config.PasswordSha256HashWO.IsNull():
		methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationSHA256Hash), Args: []string{config.PasswordSha256HashWO.ValueString()}})
	}

	if plan.Auth == nil {
		return methods
	}
	a := plan.Auth
	var ca AuthModel
	if config.Auth != nil {
		ca = *config.Auth
	}

	if a.NoPassword != nil {
		methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationNoPassword)})
	}

	methods = appendSecretMethods(methods, querybuilder.IdentificationPlaintextPassword, a.PlaintextPassword, ca.PlaintextPassword)
	methods = appendSecretMethods(methods, querybuilder.IdentificationSHA256Password, a.Sha256Password, ca.Sha256Password)
	methods = appendSha256HashMethods(methods, a.Sha256Hash, ca.Sha256Hash)
	methods = appendSecretMethods(methods, querybuilder.IdentificationDoubleSHA1Password, a.DoubleSha1Password, ca.DoubleSha1Password)
	methods = appendSecretMethods(methods, querybuilder.IdentificationDoubleSHA1Hash, a.DoubleSha1Hash, ca.DoubleSha1Hash)
	methods = appendSecretMethods(methods, querybuilder.IdentificationBcryptPassword, a.BcryptPassword, ca.BcryptPassword)
	methods = appendSecretMethods(methods, querybuilder.IdentificationBcryptHash, a.BcryptHash, ca.BcryptHash)

	for _, c := range a.SSLCertificate {
		if !c.CommonName.IsNull() {
			methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationSSLCertificateCN), Args: []string{c.CommonName.ValueString()}})
		} else {
			methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationSSLCertificateSAN), Args: []string{c.SubjectAltName.ValueString()}})
		}
	}
	for _, h := range a.HTTP {
		if !h.Server.IsNull() {
			methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationHTTPServer), Args: []string{h.Server.ValueString()}})
		} else {
			methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationHTTPScheme), Args: []string{h.Scheme.ValueString()}})
		}
	}
	for _, k := range a.SSHKey {
		methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationSSHKey), Args: []string{k.PublicKey.ValueString(), k.Type.ValueString()}})
	}
	for _, l := range a.LDAP {
		methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationLDAP), Args: []string{l.Server.ValueString()}})
	}
	for _, kb := range a.Kerberos {
		methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationKerberos), Args: []string{kb.Realm.ValueString()}})
	}

	return methods
}

func appendSecretMethods(methods []dbops.AuthMethod, typ querybuilder.Identification, plan, config []SecretMethodModel) []dbops.AuthMethod {
	for i, m := range plan {
		methods = append(methods, dbops.AuthMethod{Type: string(typ), Args: []string{secretValue(m.Value, config, i)}})
	}
	return methods
}

func appendSha256HashMethods(methods []dbops.AuthMethod, plan, config []Sha256HashModel) []dbops.AuthMethod {
	for i, m := range plan {
		value := m.Value.ValueString()
		if m.Value.IsNull() && i < len(config) {
			value = config[i].ValueWO.ValueString()
		}
		methods = append(methods, dbops.AuthMethod{Type: string(querybuilder.IdentificationSHA256Hash), Args: []string{value, m.Salt.ValueString()}})
	}
	return methods
}

func secretValue(planValue types.String, config []SecretMethodModel, i int) string {
	if !planValue.IsNull() {
		return planValue.ValueString()
	}
	if i < len(config) {
		return config[i].ValueWO.ValueString()
	}
	return ""
}
