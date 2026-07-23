You can use the `clickhousedbops_grant_privilege` resource to grant privileges on databases, tables, and parameterized ClickHouse objects to either a `clickhousedbops_user` or a `clickhousedbops_role`.

Please note that in order to grant privileges to all database and/or all tables, the `database` and/or `table` fields must be set to null, and not to "*".

For global-with-parameter privileges, use `access_object` for the parameter. This includes user and role names, named collections, table engines, and external sources. Leaving `access_object` null grants the privilege on every parameter (`ON *`).

Source `READ` and `WRITE` privileges can additionally use `access_object_filter` to restrict the source URI with a regular expression. ClickHouse treats this value as a regexp, not a literal URI, so escape regexp metacharacters such as `.` when an exact match is intended. Source filters require ClickHouse 25.8 or newer and `access_control_improvements.enable_read_write_grants`.

Known limitations:

- On ClickHouse Cloud some broad privileges (for example `ALL`, or `SELECT` on `*.*`) can't be granted directly, because the admin user holds them but can't transfer them. Set `current_grants = true` to grant them via `GRANT CURRENT GRANTS(...)`. See https://clickhouse.com/docs/en/sql-reference/statements/grant#all
- It's not possible to grant privileges using their alias name. The canonical name must be used.
- A group of privileges (such as `ALL`) can't be granted directly: grant each member of the group individually, or set `current_grants = true` to copy the group from the grantor.
- It's not possible to grant the same `clickhousedbops_grant_privilege` to both a `clickhousedbops_user` and a `clickhousedbops_role` using a single `clickhousedbops_grant_privilege` stanza. You can do that using two different stanzas, one with `grantee_user_name` and the other with `grantee_role_name` fields set.
- It's not possible to grant the same privilege (example 'SELECT') to multiple entities (for example tables) with a single stanza. You can do that my creating one stanza for each entity you want to grant privileges on.
- Importing `clickhousedbops_grant_privilege` resources into terraform is not supported.
