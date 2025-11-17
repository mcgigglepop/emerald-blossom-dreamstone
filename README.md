# vaultctl

A zero-knowledge CLI password manager with client-side encryption, built with Go, Cobra, and AWS DynamoDB.

## Features

- **Zero-knowledge architecture**: All encryption/decryption happens client-side. The server (DynamoDB) only stores encrypted blobs and never sees your master password or decrypted data.
- **Multi-device sync**: Vault can be synced across devices via DynamoDB.
- **Offline-friendly**: Vault cached locally in an encrypted file.
- **Strong cryptography**: Uses Argon2id for key derivation and XChaCha20-Poly1305 for encryption.

## Installation

### Prerequisites

- Go 1.21 or later
- AWS CLI configured (for DynamoDB access)

### Build

```bash
go mod download
go build -o vaultctl
```

### Install

```bash
go install
```

## Setup

### 1. Create DynamoDB Table

Create a DynamoDB table named `vaultctl_vaults` with:
- Partition key: `PK` (String)
- Sort key: `SK` (String)

Or use a different table name and configure it (see Configuration below).

### 2. Configure AWS Credentials

Configure AWS credentials using one of these methods:
- AWS CLI: `aws configure`
- Environment variables: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- IAM role (if running on EC2)

### 3. Initialize Vault

```bash
vaultctl init
```

This will prompt you for a master password and create an encrypted vault locally and in DynamoDB.

## Usage

### Initialize a new vault

```bash
vaultctl init
```

### Unlock the vault

```bash
vaultctl unlock
```

### Add a password entry

```bash
vaultctl add --name github --username user@example.com --url https://github.com/login --notes "2FA enabled"
```

The password will be prompted securely.

### Get a password entry

```bash
vaultctl get github
# or by ID
vaultctl get <entry-id>
```

### List all entries

```bash
vaultctl list
```

### Remove an entry

```bash
vaultctl remove github
# or by ID
vaultctl remove <entry-id>
```

### Sync with DynamoDB

```bash
vaultctl sync
```

### Create a backup

```bash
vaultctl backup
# or specify a path
vaultctl backup /path/to/backup.enc
```

### Rotate master password

```bash
vaultctl rotate-master
```

## Configuration

Configuration is stored in `~/.vaultctl/config.json`. Default values:

```json
{
  "aws_region": "us-east-1",
  "table_name": "vaultctl_vaults",
  "user_id": "default",
  "vault_path": "~/.vaultctl/vault.db"
}
```

You can edit this file directly or it will be created automatically on first use.

## Security Considerations

- **Master password**: Never logged, never stored, never sent to AWS.
- **In-memory secrets**: Secrets are kept in memory only during the CLI session.
- **Encryption**: All data is encrypted with XChaCha20-Poly1305 using keys derived from your master password via Argon2id.
- **Zero-knowledge**: DynamoDB never sees:
  - Master password
  - Master key
  - Vault key
  - Plaintext vault or entries

## Architecture

### Key Derivation

1. Master password → Argon2id KDF → Master Key
2. Master Key encrypts Vault Key (random 32-byte key)
3. Vault Key encrypts all vault entries

### Storage

- **Local**: Encrypted vault stored at `~/.vaultctl/vault.db`
- **Remote**: Encrypted vault blob stored in DynamoDB table

### Vault Format

The vault is stored as an encrypted JSON structure containing:
- Salt for master key derivation
- Encrypted vault key
- KDF parameters
- Encrypted vault ciphertext
- Version and metadata

## IAM Permissions

Minimum IAM policy for DynamoDB access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "dynamodb:GetItem",
        "dynamodb:PutItem"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/vaultctl_vaults"
    }
  ]
}
```

## Troubleshooting

### DynamoDB not available

If you see "Warning: DynamoDB not available", check:
1. AWS credentials are configured
2. AWS region is correct
3. DynamoDB table exists
4. IAM permissions are correct

The vault will still work locally without DynamoDB.

### Version conflicts

If you see "version conflict" errors, run:
```bash
vaultctl sync
```

This will sync your local vault with the remote version.

## License

MIT
