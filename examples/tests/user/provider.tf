# This file is generated automatically please do not edit
terraform {
  required_providers {
    clickhousedbops = {
      version = "0.1.0"
      source  = "ClickHouse/clickhousedbops"
    }
  }
}

provider "clickhousedbops" {
  protocol = var.protocol

  host = var.host
  port = var.port

  auth_config = {
    strategy = "password"
    username = var.username
    password = var.password
  }
}
