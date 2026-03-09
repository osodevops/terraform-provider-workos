resource "workos_permission" "billing_read" {
  slug        = "billing:read"
  name        = "Read Billing"
  description = "Allows reading billing data"
}
