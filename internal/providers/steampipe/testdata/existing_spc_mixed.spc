# User-managed custom connection - no marker
connection "my_local" {
  plugin    = "aws"
  profile   = "local-dev"
  regions   = ["us-east-1"]
}

# managed-by: cfgctl
connection "aws_old_profile" {
  plugin  = "aws"
  profile = "old-profile"
  regions = ["*"]
}

# Another user block
connection "aws_cross_account" {
  plugin  = "aws"
  regions = ["us-east-1"]
}
