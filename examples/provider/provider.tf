terraform {
  required_providers {
    clickhousedbops = {
      version = "0.1.0"
      source  = "ClickHouse/clickhousedbops"
    }
  }
}

provider "clickhousedbops" {
  host = "localhost"

  protocol = "native"
  port = 9000

  auth_config = {
    strategy = "password"
    username = "default"
    password = "changeme"
  }
}
