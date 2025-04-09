resource "clickhousedbops_role" "writer" {
 provider = clickhousedbops.native
 name = "writer"
}
