resource "clickhousedbops_role" "writer" {
  cluster_name = var.cluster_name
  name = "writer"
}
