---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "clickhousedbops_database Resource - clickhousedbops"
subcategory: ""
description: |-
  Use the clickhousedbops_database resource to create a database in a ClickHouse instance.
  Known limitations:
  Changing the comment on a database resource is unsupported and will cause the database to be destroyed and recreated. WARNING: you will lose any content of the database if you do so!
---

# clickhousedbops_database (Resource)

Use the *clickhousedbops_database* resource to create a database in a ClickHouse instance.

Known limitations:

- Changing the comment on a `database` resource is unsupported and will cause the database to be destroyed and recreated. WARNING: you will lose any content of the database if you do so!

## Example Usage

```terraform
resource "clickhousedbops_database" "logs" {
  cluster_name = "cluster"
  name = "logs"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Name of the database

### Optional

- `cluster_name` (String) Name of the cluster to create the database into. If omitted, the database will be created on the replica hit by the query.
This field must be left null when using a ClickHouse Cloud cluster.
Should be set when hitting a cluster with more than one replica.
- `comment` (String) Comment associated with the database

### Read-Only

- `uuid` (String) The system-assigned UUID for the database

## Import

Import is supported using the following syntax:

```shell
# Databases can be imported by specifying the UUID.
# Find the UUID of the database by checking system.databases table.
terraform import clickhousedbops_database.example xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

# It's also possible to import databases using the name:

terraform import clickhousedbops_database.example databasename

# IMPORTANT: if you have a multi node cluster, you need to specify the cluster name!

terraform import clickhousedbops_database.example cluster:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
terraform import clickhousedbops_database.example cluster:databasename
```
