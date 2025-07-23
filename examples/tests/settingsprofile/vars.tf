variable "protocol" {
  type = string
  default = "native"
}

variable "host" {
  type = string
  default = "localhost"
}

variable "port" {
  type = number
  default = 9000
}

variable "auth_strategy" {
  type = string
  default = "password"
}

variable "username" {
  type = string
  default = "default"
}

variable "password" {
  type = string
  default = null
}

variable "cluster_name" {
  type = string
  default = null
}
