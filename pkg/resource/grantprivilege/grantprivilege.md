You can use the `clickhousedbops_grant_privilege` resource to grant privileges on databases and tables to either a `clickhousedbops_user` or a `clickhousedbops_role`.

Please note that in order to grant privileges to all database and/or all tables, the `database` and/or `table` fields must be set to null, and not to "*".

Known limitations:

- On ClickHouse Cloud some broad privileges (for example `ALL`, or `SELECT` on `*.*`) can't be granted directly, because the admin user holds them but can't transfer them. Set `current_grants = true` to grant them via `GRANT CURRENT GRANTS(...)`. See https://clickhouse.com/docs/en/sql-reference/statements/grant#all
- It's not possible to grant privileges using their alias name. The canonical name must be used.
- A group of privileges (such as `ALL`) can't be granted directly: grant each member of the group individually, or set `current_grants = true` to copy the group from the grantor.
- It's not possible to grant the same `clickhousedbops_grant_privilege` to both a `clickhousedbops_user` and a `clickhousedbops_role` using a single `clickhousedbops_grant_privilege` stanza. You can do that using two different stanzas, one with `grantee_user_name` and the other with `grantee_role_name` fields set.
- It's not possible to grant the same privilege (example 'SELECT') to multiple entities (for example tables) with a single stanza. You can do that my creating one stanza for each entity you want to grant privileges on.
- It's not possible to grant privileges to a role when a user with the same name exists: ClickHouse resolves grantee names to users first, so the grant would silently target the user instead. The provider rejects such grants.
- Importing `clickhousedbops_grant_privilege` resources into terraform is not supported.
