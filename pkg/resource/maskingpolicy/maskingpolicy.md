You can use the `clickhousedbops_masking_policy` resource to manage a [masking policy](https://clickhouse.com/docs/cloud/guides/data-masking) on a ClickHouse table.

A masking policy rewrites the listed columns for the grantees named in the policy, optionally only for the rows matching `where_expression`. Use it to hide PII or secrets from a role while leaving the underlying data untouched.

~> **ClickHouse Cloud only**: masking policies are only available on ClickHouse Cloud (version 25.12+) and the feature must be enabled for the service. Open-source ClickHouse rejects the DDL.

Resource can be imported by `id` or the `<database>.<table>.<name>` triple.

## Grantees

A policy applies either to a specific set of grantees or to everyone. Set exactly one of:

- `grantee_names`: a list of user and role names. ClickHouse stores these as one untyped list and resolves each name to a user before a role, so users and roles are not distinguished here.
- `grantee_all_except`: apply to all users and roles, excluding the ones listed. An empty set (`[]`) applies to everyone with no exclusions.
