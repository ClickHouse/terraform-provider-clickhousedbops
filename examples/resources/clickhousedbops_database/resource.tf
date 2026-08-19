resource "clickhousedbops_database" "logs" {
  cluster_name = "cluster"
  name         = "logs"

  engine = "Replicated"
  engine_arguments = [
    "'/clickhouse/databases/{uuid}'",
    "'{shard}'",
    "'{replica}'",
  ]
}
