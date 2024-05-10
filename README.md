# `terraform-provider-jq`

This is an experimental OpenTofu and Terraform function provider based
on terraform-plugin-go.

It provides an "exec" function which takes a jq program as the first
parameter and passes all additional parameters to the function defined
in a JSON content/file.

```hcl
locals {
  jq_add = <<EOT
.foo | add
EOT
}

output "output_add" {
  value = provider::jq::exec(local.jq_add, "{\"foo\":[1,2,3,4]}")
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
```

Output:

```
Changes to Outputs:
 + output_add  = "10"
 + template    = jsonencode(
        {
          + alpha = 102
          + omega = [
              + 28,
              + 29,
              + 30,
              + 31,
            ]
        }
    )
```
