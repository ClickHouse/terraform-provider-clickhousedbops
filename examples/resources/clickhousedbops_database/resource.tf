resource "clickhousedbops_database" "logs" {
  cluster_name = "cluster"
  name = "logs"
}
