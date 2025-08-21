# This file is generated automatically please do not edit
# This file is generated automatically please do not edit
terraform {
  required_providers {
    clickhousedbops = {
      version = "1.3.1"
      source  = "ClickHouse/clickhousedbops"
    }
  }
}

provider "clickhousedbops" {
  protocol = var.protocol

  host = var.host
  port = var.port

  auth_config = {
    strategy = var.auth_strategy
    username = var.username
    password = var.password
  }
}
