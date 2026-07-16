You can use the `clickhousedbops_user` resource to create a user in a `ClickHouse` instance.

## Authentication

Use the `auth` block to configure the user's authentication. ClickHouse supports combining several
methods, so `auth` may contain multiple method blocks, and every method block except `no_password`
may be repeated:

```terraform
resource "clickhousedbops_user" "example" {
  name = "example"

  auth {
    sha256_hash {
      value_wo         = sha256("changeme")
      value_wo_version = 1
    }
    ssl_certificate {
      common_name = "example-service"
    }
  }
}
```

Supported method blocks: `no_password`, `plaintext_password`, `sha256_password`, `sha256_hash`,
`double_sha1_password`, `double_sha1_hash`, `bcrypt_password`, `bcrypt_hash`, `ssl_certificate`
(`common_name` or `subject_alt_name`), `http` (`server` or `scheme`), `ssh_key` (`public_key` +
`type`), `ldap` (`server`) and `kerberos` (optional `realm`).

- At least one authentication method must be configured.

- `no_password` is exclusive — it cannot be combined with any other method.

- The password/hash methods take a secret value; set exactly one of:

  - `value_wo` (with `value_wo_version`): write-only, never stored in state (Terraform/OpenTofu >= 1.11).
    Bump `value_wo_version` to re-apply the value.

  - `value`: stored in state, for Terraform/OpenTofu < 1.11.

## Legacy password fields (deprecated)

`password_sha256_hash` / `password_sha256_hash_wo` (with `password_sha256_hash_wo_version`) are kept
for backwards compatibility and behave as a single `sha256_hash` method. They compose additively with
the `auth` block, so existing configurations keep working — but prefer the `auth.sha256_hash` block
for new ones. Changing a legacy password field replaces the user.

Known limitations:

- Authentication values cannot be read back from ClickHouse, so external drift of a secret is not
  detected; bump the relevant `*_version` to force a re-apply of a write-only value.

- Changing a write-only value alone does not trigger an update — you must also bump its `*_version`.

- On import only the user identity is read; the configured authentication methods are re-asserted from
  configuration on the next apply.
