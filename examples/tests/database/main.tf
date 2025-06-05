resource "clickhousedbops_database" "logs" {
  cluster_name = var.cluster_name
  name         = "logs"
  comment      = "Database for logs"
}
