---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "clickhousedbops Provider"
subcategory: ""
description: |-
  
---

# clickhousedbops Provider



## Example Usage

```terraform
# This file is generated automatically please do not edit
terraform {
  required_providers {
    clickhousedbops = {
      version = "1.3.0"
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
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `auth_config` (Attributes) Authentication configuration (see [below for nested schema](#nestedatt--auth_config))
- `host` (String) The hostname to use to connect to the clickhouse instance
- `port` (Number) The port to use to connect to the clickhouse instance
- `protocol` (String) The protocol to use to connect to clickhouse instance. Valid options are: native, nativesecure, http, https

### Optional

- `tls_config` (Attributes) TLS configuration options (see [below for nested schema](#nestedatt--tls_config))

<a id="nestedatt--auth_config"></a>
### Nested Schema for `auth_config`

Required:

- `strategy` (String) The authentication method to use
- `username` (String) The username to use to authenticate to ClickHouse

Optional:

- `password` (String) The password to use to authenticate to ClickHouse


<a id="nestedatt--tls_config"></a>
### Nested Schema for `tls_config`

Optional:

- `insecure_skip_verify` (Boolean) Skip TLS cert verification when using the https protocol. This is insecure!
