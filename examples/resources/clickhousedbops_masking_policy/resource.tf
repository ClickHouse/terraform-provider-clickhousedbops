# Redact a client IP (keep the first two octets) and a log message for everyone
# holding the `analyst` role, except rows owned by the allowlisted teams.
resource "clickhousedbops_masking_policy" "logs_pii" {
  name          = "logs_pii"
  database_name = "logs_production"
  table_name    = "logs_unified_v4"

  masks = {
    "logMessage" = "'** redacted **'"
    "clientIp"   = "multiIf(empty(clientIp), clientIp, position(clientIp, ':') > 0, '** redacted **', concat(splitByChar('.', clientIp)[1], '.', splitByChar('.', clientIp)[2], '.x.x'))"
  }

  where_expression = "ownerId NOT IN ('team_internal', 'team_tests')"

  grantee_names = ["analyst"]
}

# Mask a column for everyone except the admins.
resource "clickhousedbops_masking_policy" "secrets" {
  name          = "secrets"
  database_name = "default"
  table_name    = "api_request_events_v1"

  masks = {
    "ipAddress" = "'** redacted **'"
  }

  grantee_all_except = ["admin"]
}
