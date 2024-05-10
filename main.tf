terraform {
  required_providers {
    jq = {
      source = "opentofu/jq"
    }
  }
}

locals {
  jq_add = <<EOT
.foo | add
EOT
}


output "output_add" {
  value = provider::jq::exec(local.jq_add, "{\"foo\":[1,2,3,4]}")
}

output "output_add2" {
  value = provider::jq::exec(local.jq_add, file("lib.json"))
}

locals {
  template = <<EOT
{alpha: (.foo[1] + 100),
omega: .foo | map(. + 27)}
EOT
}

output "template" {
  value = provider::jq::exec(local.template, "{\"foo\":[1,2,3,4]}")
}

output "template2" {
  value = provider::jq::exec(file("lib.jq"), "{\"foo\":[1,2,3,4]}")
}

output "template3" {
  value = provider::jq::exec(file("lib.jq"), file("lib.json"))
}
