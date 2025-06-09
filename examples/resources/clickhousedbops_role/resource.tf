resource "clickhousedbops_role" "writer" {
  cluster_name = "cluster"
  name         = "writer"
}
