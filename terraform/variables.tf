variable "aws_region" {
  description = "AWS region for resources"
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Environment name (e.g., dev, staging, prod)"
  type        = string
  default     = "dev"
}

variable "dynamodb_table_name" {
  description = "Name of the DynamoDB table for vault storage"
  type        = string
  default     = "vaultctl_vaults"
}

variable "iam_user_name" {
  description = "Name of the IAM user for vaultctl CLI access"
  type        = string
  default     = "vaultctl-user"
}

variable "enable_point_in_time_recovery" {
  description = "Enable point-in-time recovery for DynamoDB table"
  type        = bool
  default     = true
}

variable "enable_deletion_protection" {
  description = "Enable deletion protection for DynamoDB table"
  type        = bool
  default     = false
}

variable "create_s3_backup_bucket" {
  description = "Whether to create an S3 bucket for encrypted backups"
  type        = bool
  default     = false
}

variable "s3_backup_bucket_name" {
  description = "Name of the S3 bucket for backups (must be globally unique)"
  type        = string
  default     = ""
  validation {
    condition     = var.create_s3_backup_bucket ? var.s3_backup_bucket_name != "" : true
    error_message = "s3_backup_bucket_name must be provided if create_s3_backup_bucket is true."
  }
}

