resource "clickhousedbops_database" "logs" {
  cluster_name = "cluster"
  name = "logs"
}

# A Replicated database replicates DDL across the cluster. Its tables still need Replicated*MergeTree engines to replicate data.
resource "clickhousedbops_database" "events" {
  cluster_name = "cluster"
  name         = "events"
  engine       = "Replicated('/clickhouse/databases/{database}', '{shard}', '{replica}')"
}
