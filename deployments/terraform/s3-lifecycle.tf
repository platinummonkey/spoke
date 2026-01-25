# S3 Lifecycle Configuration for Spoke Storage
#
# This Terraform configuration manages lifecycle policies for the Spoke S3 bucket
# to optimize storage costs by transitioning objects to cheaper storage classes
# and expiring old data.
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply
#
# Variables:
#   - bucket_name: Name of the S3 bucket (default: spoke-storage)
#   - enable_versioning: Enable S3 versioning (default: true)
#   - backup_retention_days: Days to retain backups (default: 30)

terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Variables
variable "bucket_name" {
  description = "Name of the S3 bucket for Spoke storage"
  type        = string
  default     = "spoke-storage"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)"
  type        = string
  default     = "prod"
}

variable "enable_versioning" {
  description = "Enable S3 versioning for disaster recovery"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain database backups"
  type        = number
  default     = 30
}

variable "tags" {
  description = "Tags to apply to S3 bucket"
  type        = map(string)
  default = {
    Application = "Spoke"
    ManagedBy   = "Terraform"
  }
}

# S3 Bucket (assuming it already exists, otherwise uncomment to create)
# resource "aws_s3_bucket" "spoke_storage" {
#   bucket = var.bucket_name
#   tags   = merge(var.tags, {
#     Name        = var.bucket_name
#     Environment = var.environment
#   })
# }

# Enable S3 Versioning
resource "aws_s3_bucket_versioning" "spoke_storage" {
  bucket = var.bucket_name

  versioning_configuration {
    status = var.enable_versioning ? "Enabled" : "Suspended"
  }
}

# Lifecycle Configuration for Proto Files
resource "aws_s3_bucket_lifecycle_configuration" "spoke_storage" {
  bucket = var.bucket_name

  # Rule 1: Transition proto files to cheaper storage classes over time
  rule {
    id     = "proto-files-lifecycle"
    status = "Enabled"

    filter {
      prefix = "proto-files/"
    }

    # Transition to Standard-IA after 90 days (for infrequently accessed files)
    transition {
      days          = 90
      storage_class = "STANDARD_IA"
    }

    # Transition to Glacier after 180 days (for archival)
    transition {
      days          = 180
      storage_class = "GLACIER"
    }

    # Expire after 365 days (1 year retention)
    expiration {
      days = 365
    }

    # Clean up old versions after 30 days
    noncurrent_version_expiration {
      noncurrent_days = 30
    }

    # Transition old versions to Glacier after 7 days
    noncurrent_version_transition {
      noncurrent_days = 7
      storage_class   = "GLACIER"
    }
  }

  # Rule 2: Database backups lifecycle (shorter retention)
  rule {
    id     = "database-backups-lifecycle"
    status = "Enabled"

    filter {
      prefix = "backups/"
    }

    # Transition to Glacier immediately for cost savings
    transition {
      days          = 1
      storage_class = "GLACIER"
    }

    # Expire backups after configured retention period
    expiration {
      days = var.backup_retention_days
    }

    # Clean up old backup versions after 7 days
    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }

  # Rule 3: Compiled artifacts lifecycle
  rule {
    id     = "compiled-artifacts-lifecycle"
    status = "Enabled"

    filter {
      prefix = "compiled/"
    }

    # Transition to Standard-IA after 30 days
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    # Transition to Glacier after 90 days
    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    # Expire after 180 days (compiled artifacts can be regenerated)
    expiration {
      days = 180
    }

    # Clean up old versions after 7 days
    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }

  # Rule 4: Clean up incomplete multipart uploads
  rule {
    id     = "cleanup-incomplete-uploads"
    status = "Enabled"

    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }

  # Rule 5: Delete markers cleanup (when versioning is enabled)
  rule {
    id     = "cleanup-delete-markers"
    status = "Enabled"

    expiration {
      expired_object_delete_marker = true
    }
  }

  depends_on = [aws_s3_bucket_versioning.spoke_storage]
}

# Enable Server-Side Encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "spoke_storage" {
  bucket = var.bucket_name

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
      # For KMS encryption, use:
      # sse_algorithm     = "aws:kms"
      # kms_master_key_id = aws_kms_key.spoke.arn
    }
  }
}

# Block Public Access
resource "aws_s3_bucket_public_access_block" "spoke_storage" {
  bucket = var.bucket_name

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Bucket Policy for Secure Access
resource "aws_s3_bucket_policy" "spoke_storage" {
  bucket = var.bucket_name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "DenyInsecureTransport"
        Effect = "Deny"
        Principal = "*"
        Action = "s3:*"
        Resource = [
          "arn:aws:s3:::${var.bucket_name}",
          "arn:aws:s3:::${var.bucket_name}/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      }
    ]
  })
}

# Outputs
output "bucket_name" {
  description = "Name of the S3 bucket"
  value       = var.bucket_name
}

output "versioning_status" {
  description = "S3 versioning status"
  value       = aws_s3_bucket_versioning.spoke_storage.versioning_configuration[0].status
}

output "lifecycle_rules" {
  description = "Number of lifecycle rules configured"
  value       = length(aws_s3_bucket_lifecycle_configuration.spoke_storage.rule)
}

output "estimated_cost_savings" {
  description = "Estimated annual cost savings from lifecycle policies"
  value       = "Approximately 60-80% reduction in storage costs after transitions"
}
