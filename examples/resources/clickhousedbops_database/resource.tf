resource "clickhousedbops_database" "logs" {
  cluster_name = "cluster"
  name         = "logs"
}

resource "clickhousedbops_database" "replicated_logs" {
  cluster_name = "cluster"
  name         = "replicated_logs"

  engine = "Replicated"
  engine_arguments = [
    "'/clickhouse/databases/{uuid}'",
    "'{shard}'",
    "'{replica}'",
  ]
}
