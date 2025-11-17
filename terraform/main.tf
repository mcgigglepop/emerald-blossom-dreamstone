provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "vaultctl"
      ManagedBy   = "Terraform"
      Environment = var.environment
    }
  }
}

# DynamoDB Table for storing encrypted vaults
resource "aws_dynamodb_table" "vaults" {
  name         = var.dynamodb_table_name
  billing_mode = "PAY_PER_REQUEST" # On-demand pricing
  hash_key     = "PK"
  range_key    = "SK"

  attribute {
    name = "PK"
    type = "S"
  }

  attribute {
    name = "SK"
    type = "S"
  }

  # Point-in-time recovery for backup/restore
  point_in_time_recovery {
    enabled = var.enable_point_in_time_recovery
  }

  # Server-side encryption
  server_side_encryption {
    enabled = true
  }

  # Deletion protection (optional, set to false for easier cleanup)
  deletion_protection_enabled = var.enable_deletion_protection

  lifecycle {
    ignore_changes = [
      read_capacity,
      write_capacity,
    ]
  }

  tags = {
    Name        = "${var.dynamodb_table_name}-table"
    Description = "DynamoDB table for vaultctl encrypted vault storage"
  }
}

# IAM User for vaultctl CLI access
resource "aws_iam_user" "vaultctl" {
  name = var.iam_user_name
  path = "/vaultctl/"

  tags = {
    Name        = var.iam_user_name
    Description = "IAM user for vaultctl CLI access"
  }
}

# IAM Access Key for the user
resource "aws_iam_access_key" "vaultctl" {
  user = aws_iam_user.vaultctl.name
}

# IAM Policy for DynamoDB access
resource "aws_iam_user_policy" "vaultctl_dynamodb" {
  name = "${var.iam_user_name}-dynamodb-policy"
  user = aws_iam_user.vaultctl.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:GetItem",
          "dynamodb:PutItem",
          "dynamodb:UpdateItem",
          "dynamodb:Query"
        ]
        Resource = [
          aws_dynamodb_table.vaults.arn,
          "${aws_dynamodb_table.vaults.arn}/index/*"
        ]
      },
      {
        Effect = "Allow"
        Action = [
          "dynamodb:DescribeTable"
        ]
        Resource = aws_dynamodb_table.vaults.arn
      }
    ]
  })
}

# Optional: S3 bucket for encrypted backups
resource "aws_s3_bucket" "vault_backups" {
  count  = var.create_s3_backup_bucket ? 1 : 0
  bucket = var.s3_backup_bucket_name

  tags = {
    Name        = "${var.s3_backup_bucket_name}-backups"
    Description = "S3 bucket for vaultctl encrypted backups"
  }
}

# S3 bucket versioning for backups
resource "aws_s3_bucket_versioning" "vault_backups" {
  count  = var.create_s3_backup_bucket ? 1 : 0
  bucket = aws_s3_bucket.vault_backups[0].id

  versioning_configuration {
    status = "Enabled"
  }
}

# S3 bucket encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "vault_backups" {
  count  = var.create_s3_backup_bucket ? 1 : 0
  bucket = aws_s3_bucket.vault_backups[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# S3 bucket public access block
resource "aws_s3_bucket_public_access_block" "vault_backups" {
  count  = var.create_s3_backup_bucket ? 1 : 0
  bucket = aws_s3_bucket.vault_backups[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# IAM Policy for S3 backup access (if S3 bucket is created)
resource "aws_iam_user_policy" "vaultctl_s3" {
  count = var.create_s3_backup_bucket ? 1 : 0
  name  = "${var.iam_user_name}-s3-policy"
  user  = aws_iam_user.vaultctl.name

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = [
          aws_s3_bucket.vault_backups[0].arn,
          "${aws_s3_bucket.vault_backups[0].arn}/*"
        ]
      }
    ]
  })
}

