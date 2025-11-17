output "dynamodb_table_name" {
  description = "Name of the DynamoDB table"
  value       = aws_dynamodb_table.vaults.name
}

output "dynamodb_table_arn" {
  description = "ARN of the DynamoDB table"
  value       = aws_dynamodb_table.vaults.arn
}

output "iam_user_name" {
  description = "Name of the IAM user"
  value       = aws_iam_user.vaultctl.name
}

output "iam_user_arn" {
  description = "ARN of the IAM user"
  value       = aws_iam_user.vaultctl.arn
}

output "aws_access_key_id" {
  description = "AWS Access Key ID for the IAM user"
  value       = aws_iam_access_key.vaultctl.id
  sensitive   = true
}

output "aws_secret_access_key" {
  description = "AWS Secret Access Key for the IAM user"
  value       = aws_iam_access_key.vaultctl.secret
  sensitive   = true
}

output "s3_backup_bucket_name" {
  description = "Name of the S3 backup bucket (if created)"
  value       = var.create_s3_backup_bucket ? aws_s3_bucket.vault_backups[0].id : null
}

output "s3_backup_bucket_arn" {
  description = "ARN of the S3 backup bucket (if created)"
  value       = var.create_s3_backup_bucket ? aws_s3_bucket.vault_backups[0].arn : null
}

output "configuration_instructions" {
  description = "Instructions for configuring vaultctl"
  sensitive   = true
  value       = <<-EOT
    Configure vaultctl with the following:

    1. Set AWS credentials:
       export AWS_ACCESS_KEY_ID="${aws_iam_access_key.vaultctl.id}"
       export AWS_SECRET_ACCESS_KEY="${aws_iam_access_key.vaultctl.secret}"
       export AWS_DEFAULT_REGION="${var.aws_region}"

    2. Or use AWS CLI:
       aws configure
       (Enter the Access Key ID and Secret Access Key from outputs)

    3. Update vaultctl config.json:
       {
         "aws_region": "${var.aws_region}",
         "table_name": "${aws_dynamodb_table.vaults.name}",
         "user_id": "default"
       }

    4. Initialize your vault:
       vaultctl init
  EOT
}

