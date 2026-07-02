You can use the `clickhousedbops_masking_policy` resource to manage a [masking policy](https://clickhouse.com/docs/cloud/guides/data-masking) on a ClickHouse table.

A masking policy rewrites the listed columns for the grantees named in the policy, optionally only for the rows matching `where_expression`. Use it to hide PII or secrets from a role while leaving the underlying data untouched.

~> **ClickHouse Cloud only**: masking policies are only available on ClickHouse Cloud (version 25.12+) and the feature must be enabled for the service. Open-source ClickHouse rejects the DDL.

~> **Read-back limitation**: only the existence of the policy is read back from `SHOW MASKING POLICIES`. The masking expressions, `where_expression` and grantees are taken from configuration, so drift made outside Terraform on those fields is not detected. Changing the `name`, `database_name` or `table_name` forces a new policy.
