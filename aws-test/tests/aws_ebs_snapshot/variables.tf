variable "resource_name" {
  type        = string
  default     = "turbot-test-20200125-create-update"
  description = "Name of the resource used throughout the test."
}

variable "aws_profile" {
  type        = string
  default     = "integration-tests"
  description = "AWS credentials profile used for the test. Default is to use the default profile."
}

variable "aws_region" {
  type        = string
  default     = "us-east-1"
  description = "AWS region used for the test. Does not work with default region in config, so must be defined here."
}

variable "aws_region_alternate" {
  type        = string
  default     = "us-east-2"
  description = "Alternate AWS region used for tests that require two regions (e.g. DynamoDB global tables)."
}

provider "aws" {
  profile = var.aws_profile
  region  = var.aws_region
}

provider "aws" {
  alias   = "alternate"
  profile = var.aws_profile
  region  = var.aws_region_alternate
}

data "aws_partition" "current" {}
data "aws_caller_identity" "current" {}
data "aws_region" "primary" {}
data "aws_region" "alternate" {
  provider = aws.alternate
}

data "null_data_source" "resource" {
  inputs = {
    scope = "arn:${data.aws_partition.current.partition}:::${data.aws_caller_identity.current.account_id}"
  }
}

resource "aws_ebs_volume" "test_volume" {
  availability_zone = "${var.aws_region}a"
  size              = 1
}

resource "aws_ebs_snapshot" "named_test_resource" {
  volume_id   = aws_ebs_volume.test_volume.id
  description = "Test snapshot"
  tags = {
    Name = var.resource_name
  }
}

resource "aws_snapshot_create_volume_permission" "example_perm" {
  snapshot_id = aws_ebs_snapshot.named_test_resource.id
  account_id  = "388460667113"
}

output "resource_aka" {
  value = aws_ebs_snapshot.named_test_resource.arn
}

output "volume_id" {
  value = aws_ebs_volume.test_volume.id
}

output "snapshot_id" {
  value = aws_ebs_snapshot.named_test_resource.id
}

output "resource_name" {
  value = var.resource_name
}

output "aws_region" {
  value = data.aws_region.primary.name
}

output "aws_partition" {
  value = data.aws_partition.current.partition
}

output "account_id" {
  value = data.aws_caller_identity.current.account_id
}
