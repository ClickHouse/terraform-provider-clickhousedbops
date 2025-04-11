You can use the `clickhousedbops_grant_role` resource to grant a `clickhousedbops_role` to either a `clickhousedbops_user` or to another `clickhousedbops_role`.

Known limitations:

- It's not possible to grant the same `clickhousedbops_role` to both a `clickhousedbops_user` and a `clickhousedbops_role` using a single `clickhousedbops_grant_role` stanza. You can do that using two different stanzas, one with `grantee_user_name` and the other with `grantee_role_name` fields set.
- Importing `clickhousedbops_grant_role` resources into terraform is not supported.
