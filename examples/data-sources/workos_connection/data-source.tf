# Look up an existing SSO connection

# Look up by connection ID
data "workos_connection" "by_id" {
  id = "conn_01HXYZ..."
}

output "connection_name_by_id" {
  value = data.workos_connection.by_id.name
}

# Look up by organization and connection type
data "workos_connection" "by_org_type" {
  organization_id = "org_01HXYZ..."
  connection_type = "OktaSAML"
}

output "connection_state" {
  value = data.workos_connection.by_org_type.state
}
