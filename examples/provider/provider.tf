# This file is generated automatically please do not edit
terraform {
  required_providers {
    clickhousedbops = {
      version = "1.12.0"
      source  = "ClickHouse/clickhousedbops"
    }
  }
}

variable "clickhouse_password" {
  type      = string
  sensitive = true
  ephemeral = true
}

provider "clickhousedbops" {
  host = "localhost"

  protocol = "native"
  port = 9000

  auth_config = {
    strategy = "password"
    username = "default"
    password = var.clickhouse_password
  }
}
