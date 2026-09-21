Use `clickhousedbops_revoke_privilege` to manage a ClickHouse partial revoke: a
negative access-right entry that removes a narrower privilege from an existing
broader grant.

For example, a role can retain `SELECT` on every column of a table except a
sensitive column:

```terraform
resource "clickhousedbops_grant_privilege" "table_reader" {
  privilege_name    = "SELECT"
  database_name     = "analytics"
  table_name        = "events"
  grantee_role_name = "analyst"
}

resource "clickhousedbops_revoke_privilege" "hide_payload" {
  privilege_name    = "SELECT"
  database_name     = "analytics"
  table_name        = "events"
  column_name       = "secret_payload"
  grantee_role_name = "analyst"

  depends_on = [clickhousedbops_grant_privilege.table_reader]

  # If the enclosing grant is replaced, its broad REVOKE can also clear this
  # partial revoke. Replacing this resource reapplies it after the grant.
  lifecycle {
    replace_triggered_by = [clickhousedbops_grant_privilege.table_reader]
  }
}
```

The resource owns a row with `is_partial_revoke = 1` in `system.grants`. It is
not equivalent to removing a `clickhousedbops_grant_privilege` resource: the
broader positive grant remains.

Set `grant_option_only = true` to manage `REVOKE GRANT OPTION FOR`, which keeps
the privilege itself but prevents the grantee from granting it to others.

On deletion, the provider grants the narrow target back. This is ClickHouse's
native operation for cancelling a partial revoke. Before doing so, the provider
verifies that a positive grant still covers the target. It fails without issuing
`GRANT` if the broader grant has already been removed, preventing deletion from
creating a standalone privilege.

Known limitations:

- A matching broader grant must exist before this resource is created.
- Keep an explicit dependency on the broader grant so Terraform destroys this
  resource first. If the broader grant is removed first, deletion fails closed
  until the grant is restored or the now-obsolete partial revoke is removed
  outside Terraform.
- Source grants (`READ ON S3`, `WRITE ON URL`, and similar) do not support
  partial revokes in ClickHouse.
- Privilege groups such as `ALL` are not accepted because ClickHouse can expand
  them into multiple scope-dependent rows. Declare one resource per canonical
  leaf privilege.
- Independently managed overlapping partial revokes are rejected because
  ClickHouse normalizes them into fewer negative access-right rows.
- Import is not supported because a partial revoke has no stable server-side ID
  and its identity includes the grantee type and complete privilege target.
