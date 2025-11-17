# Terraform Infrastructure for vaultctl

This directory contains Terraform configuration to deploy the AWS infrastructure needed for vaultctl.

## Prerequisites

- [Terraform](https://www.terraform.io/downloads) >= 1.0 installed
- AWS CLI configured with appropriate credentials
- AWS account with permissions to create:
  - DynamoDB tables
  - IAM users and policies
  - S3 buckets (optional)

## Quick Start

1. **Copy the example variables file:**
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   ```

2. **Edit `terraform.tfvars` with your values:**
   ```hcl
   aws_region = "us-east-1"
   dynamodb_table_name = "vaultctl_vaults"
   iam_user_name = "vaultctl-user"
   ```

3. **Initialize Terraform:**
   ```bash
   cd terraform
   terraform init
   ```

4. **Review the plan:**
   ```bash
   terraform plan
   ```

5. **Apply the configuration:**
   ```bash
   terraform apply
   ```

6. **Get the outputs (including AWS credentials):**
   ```bash
   terraform output
   ```

## Resources Created

### DynamoDB Table
- **Name**: `vaultctl_vaults` (configurable)
- **Billing**: Pay-per-request (on-demand)
- **Encryption**: Server-side encryption enabled
- **Point-in-time recovery**: Enabled by default
- **Schema**:
  - Partition Key: `PK` (String)
  - Sort Key: `SK` (String)

### IAM User
- **Name**: `vaultctl-user` (configurable)
- **Permissions**: 
  - DynamoDB: GetItem, PutItem, UpdateItem, Query, DescribeTable
  - S3 (if backup bucket created): PutObject, GetObject, DeleteObject, ListBucket

### S3 Backup Bucket (Optional)
- Created only if `create_s3_backup_bucket = true`
- Versioning enabled
- Server-side encryption enabled
- Public access blocked

## Configuration Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `aws_region` | AWS region for resources | `us-east-1` |
| `environment` | Environment name | `dev` |
| `dynamodb_table_name` | DynamoDB table name | `vaultctl_vaults` |
| `iam_user_name` | IAM user name | `vaultctl-user` |
| `enable_point_in_time_recovery` | Enable PITR for DynamoDB | `true` |
| `enable_deletion_protection` | Protect DynamoDB from deletion | `false` |
| `create_s3_backup_bucket` | Create S3 bucket for backups | `false` |
| `s3_backup_bucket_name` | S3 bucket name (must be unique) | `""` |

## Outputs

After applying, Terraform will output:
- DynamoDB table name and ARN
- IAM user name and ARN
- AWS Access Key ID and Secret Access Key (sensitive)
- S3 bucket name and ARN (if created)
- Configuration instructions

## Setting Up vaultctl

After deploying the infrastructure:

1. **Get your AWS credentials from Terraform outputs:**
   ```bash
   terraform output -json
   ```

2. **Configure AWS CLI:**
   ```bash
   aws configure
   # Enter the Access Key ID and Secret Access Key from outputs
   # Region: (from your terraform.tfvars)
   ```

3. **Or set environment variables:**
   ```bash
   export AWS_ACCESS_KEY_ID="<from terraform output>"
   export AWS_SECRET_ACCESS_KEY="<from terraform output>"
   export AWS_DEFAULT_REGION="<your region>"
   ```

4. **Update vaultctl config.json:**
   ```json
   {
     "aws_region": "<your region>",
     "table_name": "<dynamodb_table_name from output>",
     "user_id": "default"
   }
   ```

5. **Initialize your vault:**
   ```bash
   vaultctl init
   ```

## Security Best Practices

1. **Store credentials securely:**
   - Never commit `terraform.tfvars` or outputs to version control
   - Use AWS Secrets Manager or similar for production
   - Rotate access keys regularly

2. **IAM Permissions:**
   - The IAM user has minimal permissions (least privilege)
   - Only DynamoDB and S3 (if enabled) access
   - No administrative permissions

3. **DynamoDB:**
   - Point-in-time recovery enabled for backup/restore
   - Server-side encryption enabled
   - Consider enabling deletion protection for production

4. **S3 (if used):**
   - Versioning enabled
   - Encryption enabled
   - Public access blocked

## Cost Estimation

- **DynamoDB**: Pay-per-request pricing
  - Write: $1.25 per million requests
  - Read: $0.25 per million requests
  - Storage: $0.25 per GB-month
  - Typical usage: < $1/month for personal use

- **IAM**: Free

- **S3** (if enabled):
  - Storage: $0.023 per GB-month
  - Requests: Minimal cost
  - Typical usage: < $0.10/month for backups

## Cleanup

To destroy all resources:

```bash
terraform destroy
```

**Warning**: This will delete:
- DynamoDB table and all data
- IAM user and access keys
- S3 bucket and all backups (if created)

Make sure you have backups before destroying!

## Troubleshooting

### Error: "Table already exists"
- The table name must be unique in your AWS account
- Change `dynamodb_table_name` in `terraform.tfvars`

### Error: "Bucket name already exists"
- S3 bucket names must be globally unique
- Change `s3_backup_bucket_name` in `terraform.tfvars`

### Error: "Access Denied"
- Ensure your AWS credentials have permissions to create:
  - DynamoDB tables
  - IAM users and policies
  - S3 buckets (if enabled)

### Error: "Invalid region"
- Verify the `aws_region` variable is a valid AWS region

## Advanced Configuration

### Using Existing IAM User

If you want to use an existing IAM user instead of creating one:

1. Comment out the IAM user resources in `main.tf`
2. Manually attach the policy from `aws_iam_user_policy.vaultctl_dynamodb.policy`
3. Use your existing access keys

### Multi-Region Deployment

To deploy in multiple regions:

1. Create separate Terraform workspaces or directories per region
2. Use different `aws_region` values
3. Use different table names per region

### Custom Tags

Tags are automatically applied to all resources. To add custom tags, modify the `default_tags` block in `main.tf`.

## Support

For issues with the Terraform configuration:
- Check Terraform documentation: https://www.terraform.io/docs
- Check AWS provider documentation: https://registry.terraform.io/providers/hashicorp/aws/latest/docs

For vaultctl application issues, see the main README.md or HOW_TO_USE.txt.

