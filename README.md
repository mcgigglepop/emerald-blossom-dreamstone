# vaultctl

A zero-knowledge CLI password manager with client-side encryption. All encryption and decryption happens locally. The server (DynamoDB) only stores encrypted blobs and never sees your master password or decrypted data.

## Platform Support

vaultctl works on **Windows**, **macOS**, and **Linux**. The Go binary is cross-platform and uses standard libraries for file paths, terminal input, and OS operations.

**Note:** The `run.sh` script is Unix-specific (bash) and only works on macOS and Linux. On Windows, you can build and run the binary directly, or use Windows Subsystem for Linux (WSL) to use the script.

## Features

- **Zero-knowledge architecture**: All encryption/decryption happens client-side
- **Multi-device sync**: Vault can be synced across devices via DynamoDB
- **Offline-friendly**: Vault cached locally in an encrypted file
- **Strong cryptography**: Uses Argon2id for key derivation and XChaCha20-Poly1305 for encryption
- **Session-based unlocking**: Enter master password once per terminal session (30 minutes)
- **Backup codes support**: Store 2FA/authenticator backup codes with password entries
- **Terraform deployment**: Automated AWS infrastructure provisioning

## Installation

### Prerequisites

- Go 1.23 or later installed
- Terraform 1.0 or later installed (for AWS infrastructure deployment)
- AWS CLI configured (for Terraform and vaultctl access)
- AWS account with permissions to create DynamoDB tables, IAM users, and Secrets Manager secrets

**Platform-specific notes:**
- **Windows:** Use PowerShell or Command Prompt. Use `run.bat` or `run.ps1` scripts (included). The `run.sh` script requires WSL or Git Bash.
- **macOS/Linux:** Full support including the `run.sh` script.

### Build the Application

**OPTION A: Manual Build**

```bash
cd /path/to/emerald-blossom-dreamstone
go mod download
go build -o vaultctl
```

This creates an executable named "vaultctl" in the current directory.

**OPTION B: Using run.sh Script (macOS/Linux only)**

The project includes a convenient `run.sh` script that automates building and running the application. **This script only works on Unix-like systems (macOS, Linux, or WSL on Windows).**

First, make the script executable:

```bash
chmod +x run.sh
```

Then use it to build:

```bash
./run.sh build
```

The script will:
- Check prerequisites (Go, AWS CLI)
- Download dependencies
- Build the application
- Provide colored status messages

You can also use it to run commands directly:

```bash
./run.sh run init
./run.sh run add --name github --username user@example.com
./run.sh run list
```

For more information about run.sh:

```bash
./run.sh help
```

**Windows users:** On Windows, you can use:
- **Option A (Manual Build):** Build directly with `go build -o vaultctl.exe`
- **Option B (Windows Scripts):** Use `run.bat` (batch file) or `run.ps1` (PowerShell script) - see below
- **WSL:** Use WSL (Windows Subsystem for Linux) to run the `run.sh` script

**Windows Scripts:**

The project includes Windows equivalents of the `run.sh` script:

- **`run.bat`** - Batch file (works on all Windows versions)
  ```cmd
  run.bat build
  run.bat run init
  run.bat run add --name github --username user@example.com
  ```

- **`run.ps1`** - PowerShell script (requires PowerShell, better features)
  ```powershell
  .\run.ps1 build
  .\run.ps1 run init
  .\run.ps1 run add --name github --username user@example.com
  ```

  **Note:** If you get an execution policy error with PowerShell, run:
  ```powershell
  Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
  ```

### Install (Optional)

You can move the vaultctl binary to a directory in your PATH:

```bash
sudo mv vaultctl /usr/local/bin/
```

Or install via Go:

```bash
go install
```

## Initial Setup

### STEP 1: Deploy AWS Infrastructure with Terraform

The project includes Terraform configurations to automatically create all required AWS resources (DynamoDB table, IAM user, Secrets Manager secret).

1. Navigate to the terraform directory:

```bash
cd terraform
```

2. (Optional) Customize variables:

Edit `terraform.tfvars` to set your preferred values:
- `aws_region`: AWS region for resources (default: us-west-2)
- `dynamodb_table_name`: DynamoDB table name (default: vaultctl_vaults)
- `iam_user_name`: IAM user name (default: vaultctl-user)
- `create_s3_backup_bucket`: Whether to create S3 bucket for backups (default: false)

Example `terraform.tfvars`:

```hcl
aws_region  = "us-west-2"
environment = "dev"
dynamodb_table_name = "vaultctl_vaults"
iam_user_name = "vaultctl-user"
```

3. Initialize Terraform:

```bash
terraform init
```

4. Review the deployment plan:

```bash
terraform plan
```

5. Deploy the infrastructure:

```bash
terraform apply
```

Terraform will create:
- DynamoDB table with PK (String) and SK (String) keys
- IAM user with appropriate permissions
- AWS Secrets Manager secret for session key encryption
- (Optional) S3 bucket for backups

6. Save the Terraform outputs:

After deployment, Terraform will display important outputs including:
- AWS Access Key ID and Secret Access Key for the IAM user
- DynamoDB table name
- Secrets Manager secret name
- Configuration instructions

Save these outputs securely - you'll need them in the next step.

### STEP 2: Configure AWS Credentials

Use the credentials from Terraform outputs:

**A) Environment Variables (Recommended):**

```bash
export AWS_ACCESS_KEY_ID=<access_key_from_terraform_output>
export AWS_SECRET_ACCESS_KEY=<secret_key_from_terraform_output>
export AWS_DEFAULT_REGION=<region_from_terraform_output>
```

To get the credentials from Terraform:

```bash
terraform output -raw aws_access_key_id
terraform output -raw aws_secret_access_key
terraform output aws_region
```

**B) AWS CLI Configuration:**

```bash
aws configure
```

Enter the Access Key ID and Secret Access Key from Terraform outputs. Enter the region (should match your Terraform deployment).

**C) IAM Role (if running on EC2):**

Attach an IAM role with the same permissions as the Terraform-created user.

**Note:** The Terraform-created IAM user already has all necessary permissions for DynamoDB, Secrets Manager, and (optionally) S3. No manual IAM policy configuration is needed.

### STEP 3: Configure vaultctl

After Terraform deployment, update your vaultctl configuration:

1. Get the configuration values from Terraform:

```bash
terraform output configuration_instructions
```

2. Edit `~/.vaultctl/config.json` (created on first use):

```json
{
  "aws_region": "<region_from_terraform>",
  "table_name": "<table_name_from_terraform>",
  "user_id": "default",
  "session_secret_name": "vaultctl/session-key"
}
```

The `session_secret_name` should be "vaultctl/session-key" (default from Terraform).

### STEP 4: Initialize Your Vault

Run the init command:

```bash
./run.sh run init
# Or: vaultctl init
```

You will be prompted to:
1. Enter a master password (this encrypts your vault)
2. Confirm the master password

**IMPORTANT:** Remember your master password! If you lose it, you cannot recover your vault. The master password is never stored anywhere.

After initialization, your vault is created locally at:
- `~/.vaultctl/vault.db`

And synced to DynamoDB (if configured).

**Note:** If using run.sh for the first time, it will automatically build the application before running the init command.

## Basic Usage

You can use vaultctl in two ways:
1. Using the run.sh script: `./run.sh run [command]`
2. Directly: `vaultctl [command]` (after building)

Both methods work the same way. The run.sh script is convenient because it automatically builds the application if needed.

### Unlock the Vault

Before using most commands, you need to unlock the vault:

```bash
./run.sh run unlock
# Or: vaultctl unlock
```

Enter your master password when prompted. The vault stays unlocked for the duration of your terminal session (30 minutes by default).

### Session-Based Unlocking

vaultctl uses session-based unlocking, which means you only need to enter your master password **ONCE** per terminal session. After unlocking:

- All subsequent commands (add, get, list, remove, etc.) work automatically
- No password prompt needed for 30 minutes
- Session persists across different terminal windows in the same session
- Session expires after 30 minutes of inactivity

**How it works:**
1. First command: Run `vaultctl unlock` and enter master password
2. Session created: Vault key is encrypted and stored in `~/.vaultctl/session.json`
3. Subsequent commands: Automatically use the session (no password needed)
4. Session expires: After 30 minutes, you'll be prompted for password again

### Lock the Vault

To manually lock the vault and clear the session:

```bash
./run.sh run lock
# Or: vaultctl lock
```

This clears the session and requires you to unlock again for the next command.

**Note:** Some commands (like add, get, list) will automatically prompt you to unlock if the vault is locked or the session has expired.

### Add a Password Entry

Add a new password entry:

```bash
./run.sh run add --name github --username user@example.com --url https://github.com/login --notes "2FA enabled"
# Or: vaultctl add --name github --username user@example.com --url https://github.com/login --notes "2FA enabled"
```

**Flags:**
- `--name` (required) Name/identifier for the entry
- `--username` (optional) Username or email
- `--url` (optional) Website URL
- `--notes` (optional) Additional notes
- `--backup-codes` (optional) 2FA/authenticator backup codes (comma or semicolon separated)
- `--no-sync` (optional) Don't sync to DynamoDB after adding

The password will be prompted securely (hidden input).

If you don't provide `--backup-codes`, you'll be asked if you want to add backup codes interactively. This is useful for storing authenticator backup codes securely.

**Example:**

```bash
./run.sh run add --name "Work Email" --username "john@company.com"
# Or: vaultctl add --name "Work Email" --username "john@company.com"
# Enter password: [hidden]
# Enter backup codes? (y/n, or press Enter to skip): y
# Enter backup codes (one per line, empty line to finish):
#   Code: ABC123-XYZ789
#   Code: DEF456-UVW012
#   Code: [Enter to finish]
```

**Example with backup codes flag:**

```bash
vaultctl add --name "GitHub" --username "dev@example.com" \
  --backup-codes "ABC123-XYZ789,DEF456-UVW012,GHI789-RST345"
```

### Get a Password Entry

Retrieve a password entry by name or ID:

```bash
./run.sh run get github
./run.sh run get <entry-id>
# Or: vaultctl get github
# Or: vaultctl get <entry-id>
```

This displays all information for the entry including:
- Name, username, password
- URL and notes
- Backup codes (if any)
- Created and updated timestamps

### List All Entries

List all entries without showing passwords:

```bash
./run.sh run list
# Or: vaultctl list
```

Output shows: Name, Username, URL, Last Updated

### Update an Entry

Update fields of an existing entry:

```bash
./run.sh run update github --username "newuser@example.com"
./run.sh run update <entry-id> --backup-codes "CODE1,CODE2,CODE3"
# Or: vaultctl update github --username "newuser@example.com"
# Or: vaultctl update github --backup-codes "CODE1,CODE2,CODE3"
```

**Flags:**
- `--name` (optional) Update entry name
- `--username` (optional) Update username
- `--password` (optional) Update password (leave empty to prompt securely)
- `--url` (optional) Update URL
- `--notes` (optional) Update notes
- `--backup-codes` (optional) Update backup codes (comma/semicolon separated, or empty string to clear)
- `--no-sync` (optional) Don't sync to DynamoDB after updating

Only the fields you specify will be updated. Other fields remain unchanged.

**Examples:**

```bash
# Update username only
vaultctl update github --username "newemail@example.com"

# Add backup codes to existing entry
vaultctl update github --backup-codes "ABC123,DEF456,GHI789"

# Clear backup codes
vaultctl update github --backup-codes ""

# Update multiple fields
vaultctl update github --username "newuser" --url "https://newurl.com" --notes "Updated"
```

### Remove an Entry

Delete an entry:

```bash
./run.sh run remove github
./run.sh run remove <entry-id>
# Or: vaultctl remove github
# Or: vaultctl remove <entry-id>
```

**Flags:**
- `--no-sync` (optional) Don't sync to DynamoDB after removing

### Sync with DynamoDB

Sync your local vault with the remote DynamoDB vault:

```bash
./run.sh run sync
# Or: vaultctl sync
```

This will:
- Pull the latest version from DynamoDB if it's newer
- Push your local changes if they're newer
- Handle version conflicts

If there's a conflict, you'll be prompted to resolve it.

### Create a Backup

Create an encrypted backup of your vault:

```bash
./run.sh run backup
# Or: vaultctl backup
```

This creates a backup file at:
- `~/.vaultctl/backups/vault-YYYY-MM-DDTHH-MM-SSZ.enc`

Or specify a custom path:

```bash
./run.sh run backup /path/to/backup.enc
# Or: vaultctl backup /path/to/backup.enc
```

The backup is encrypted with the same encryption as your vault.

### Restore from Backup

Restore your vault from a backup file:

```bash
./run.sh run restore
# Or: vaultctl restore
```

If no backup path is provided, the command will:
1. List all available backups in `~/.vaultctl/backups/`
2. Display backup details (filename, creation date, size)
3. Prompt you to select which backup to restore
4. Optionally create a backup of your current vault before restoring
5. Replace your vault.db with the selected backup

You can also restore directly from a specific backup file:

```bash
./run.sh run restore /path/to/backup.enc
# Or: vaultctl restore /path/to/backup.enc
```

**IMPORTANT:** Restoring will replace your current vault. Make sure you have a backup of your current vault if needed. The restore command will offer to create a backup of your current vault before restoring.

### Rotate Master Password

Change your master password:

```bash
./run.sh run rotate-master
# Or: vaultctl rotate-master
```

You will be prompted to:
1. Enter your current master password
2. Enter your new master password
3. Confirm your new master password

This re-encrypts your vault key with the new master password. Your entries remain unchanged.

## Configuration

Configuration is stored at: `~/.vaultctl/config.json`

**Default configuration:**

```json
{
  "aws_region": "us-west-2",
  "table_name": "vaultctl_vaults",
  "user_id": "default",
  "vault_path": "~/.vaultctl/vault.db",
  "session_secret_name": "vaultctl/session-key"
}
```

You can edit this file directly to change:
- AWS region (should match your Terraform deployment)
- DynamoDB table name (should match your Terraform deployment)
- User ID (for multi-user scenarios)
- Local vault file path
- Session secret name (should match your Terraform deployment, default: "vaultctl/session-key")

The config file is created automatically on first use. After deploying with Terraform, update it with the values from your Terraform outputs.

## Examples

### Example 1: Complete Workflow (Using run.sh)

1. Deploy AWS infrastructure with Terraform:

```bash
cd terraform
terraform init
terraform plan
terraform apply

# Save the Terraform outputs (especially AWS credentials):
terraform output -raw aws_access_key_id
terraform output -raw aws_secret_access_key
terraform output configuration_instructions
```

2. Configure AWS credentials:

```bash
export AWS_ACCESS_KEY_ID=<from_terraform_output>
export AWS_SECRET_ACCESS_KEY=<from_terraform_output>
export AWS_DEFAULT_REGION=<from_terraform_output>
```

3. Build the application:

```bash
cd ..  # Return to project root
./run.sh build
```

4. Configure vaultctl (update config.json with Terraform values):

Edit `~/.vaultctl/config.json` with values from `terraform output configuration_instructions`

5. Initialize vault:

```bash
./run.sh run init
# Enter master password: ********
# Confirm master password: ********
# Vault initialized and synced to DynamoDB
```

6. Unlock the vault (only needed once per session):

```bash
./run.sh run unlock
# Enter master password: ********
# Vault unlocked successfully
```

7. Add entries (no master password prompt needed - using session):

```bash
./run.sh run add --name "Gmail" --username "me@gmail.com" --url "https://gmail.com"
# Enter password: ********  (this is the entry password, not master password)
# Enter backup codes? (y/n, or press Enter to skip): n
# Entry 'Gmail' added successfully

./run.sh run add --name "GitHub" --username "developer" --url "https://github.com" \
  --backup-codes "ABC123-XYZ789,DEF456-UVW012"
# Enter password: ********  (entry password)
# Entry 'GitHub' added successfully
```

8. List entries (no password needed):

```bash
./run.sh run list
# NAME    USERNAME    URL                  UPDATED
# Gmail   me@gmail.com https://gmail.com    2025-01-16 10:30:00
# GitHub  developer   https://github.com    2025-01-16 10:31:00
```

9. Get an entry (no password needed):

```bash
./run.sh run get GitHub
# Name: GitHub
# Username: developer
# Password: mypassword123
# URL: https://github.com
# Backup Codes:
#   1. ABC123-XYZ789
#   2. DEF456-UVW012
# Created: 2025-01-16 10:30:00
# Updated: 2025-01-16 10:30:00
```

10. Update an entry to add backup codes:

```bash
./run.sh run update Gmail --backup-codes "CODE1,CODE2,CODE3"
# Entry 'Gmail' updated successfully
```

11. Sync with remote (no password needed):

```bash
./run.sh run sync
# Vault synced successfully (version 3)
```

**Note:** After unlocking once, all subsequent commands work without prompting for the master password for 30 minutes. The session persists across commands.

### Example 2: Working Offline

If DynamoDB is not available, you can still use vaultctl locally:

1. Unlock vault (only once per session):

```bash
./run.sh run unlock
# Enter master password: ********
# Vault unlocked successfully
```

2. Add entries (no master password needed - session active):

```bash
./run.sh run add --name "Local Entry" --no-sync
# Enter password: ********  (entry password only)
```

3. When DynamoDB is available again (no password needed):

```bash
./run.sh run sync
```

### Example 3: Backup and Restore

1. Create backup:

```bash
./run.sh run backup
# Backup created at: ~/.vaultctl/backups/vault-2025-01-16T10-00-00Z.enc
```

2. List and restore from backup:

```bash
./run.sh run restore
# Available backups:
# 
#   1. vault-2025-01-16T10-00-00Z.enc
#      Created: 2025-01-16 10:00:00
#      Size: 2.5 KB
# 
#   2. vault-2025-01-15T14-30-00Z.enc
#      Created: 2025-01-15 14:30:00
#      Size: 2.3 KB
# 
# Select backup to restore (enter number): 1
# Selected: vault-2025-01-16T10-00-00Z.enc
# Current vault exists. Create a backup before restoring? (y/n): y
# Current vault backed up to: ~/.vaultctl/backups/vault-before-restore-2025-01-16T10-05-00Z.enc
# Vault restored successfully from: vault-2025-01-16T10-00-00Z.enc
# You can now unlock the vault with: vaultctl unlock
```

3. Restore directly from a specific backup file:

```bash
./run.sh run restore /path/to/backup.enc
```

## Troubleshooting

### PROBLEM: "DynamoDB not available" warning

**SOLUTION:**
- Check AWS credentials are configured: `aws configure list`
- Verify AWS region matches your Terraform deployment region
- Ensure Terraform deployment completed successfully: `terraform show`
- Verify DynamoDB table exists: `aws dynamodb describe-table --table-name vaultctl_vaults`
- Check IAM permissions (should be automatic if using Terraform-created user)
- Verify config.json has correct table_name and aws_region

**Note:** The vault works locally even without DynamoDB. You just won't have cloud sync until DynamoDB is configured. If you haven't deployed with Terraform yet, run `terraform apply` in the terraform directory.

### PROBLEM: "version conflict" error

**SOLUTION:**
- Run: `vaultctl sync`
- This will sync your local vault with the remote version
- If conflicts persist, you may need to manually resolve by choosing which version to keep

### PROBLEM: "vault not found" error

**SOLUTION:**
- Run: `vaultctl init`
- This creates a new vault
- If you had an existing vault, check the vault_path in config.json

### PROBLEM: "failed to unlock vault" error

**SOLUTION:**
- Verify you're using the correct master password
- Check that the vault file exists at the configured path
- Ensure you have read permissions on the vault file
- If vault is corrupted, restore from backup

### PROBLEM: "failed to decrypt" error

**SOLUTION:**
- Verify master password is correct
- Check that vault file is not corrupted
- Try restoring from a backup if available
- Ensure you're using the same vault file that was encrypted
- If using session, try clearing it: `vaultctl lock`, then unlock again

### PROBLEM: "session expired" or "no active session" error

**SOLUTION:**
- This is normal after 30 minutes of inactivity
- Simply run: `vaultctl unlock`
- Enter your master password to create a new session
- Sessions expire for security - this is expected behavior

### PROBLEM: Session not persisting across commands

**SOLUTION:**
- Ensure you're in the same terminal session (same shell)
- Check that `~/.vaultctl/session.json` exists and has correct permissions
- Try unlocking again: `vaultctl unlock`

### PROBLEM: Command not found

**SOLUTION:**
- Ensure vaultctl is in your PATH, or
- Use full path: `/path/to/vaultctl [command]`
- Or run from the build directory: `./vaultctl [command]`

## Security Best Practices

### 1. Master Password
- Use a strong, unique master password
- Consider using a passphrase (multiple words)
- Never share your master password
- Store it securely (password manager, physical safe, etc.)

### 2. Vault File
- The vault file is encrypted, but still protect it
- Don't share the vault file
- Regular backups are recommended
- Store backups in secure locations

### 3. AWS Credentials
- Use IAM roles when possible (more secure than access keys)
- Rotate access keys regularly (Terraform outputs new keys on each apply)
- Terraform creates IAM user with least-privilege policies automatically
- Never commit AWS credentials to version control
- Store Terraform outputs securely (they contain sensitive credentials)

### 4. Environment
- Run vaultctl on trusted machines only
- Be cautious when using on shared systems
- Clear terminal history if passwords were displayed
- Lock your screen when not using the terminal
- Use `vaultctl lock` when finished working to clear the session
- Sessions expire after 30 minutes for security

### 5. Backups
- Create regular backups: `vaultctl backup`
- Store backups in multiple secure locations
- Test backup restoration periodically
- Encrypt backup storage if storing in cloud

## File Locations

Default file locations:

**macOS/Linux:**
- **Configuration:** `~/.vaultctl/config.json`
- **Vault file:** `~/.vaultctl/vault.db`
- **Session file:** `~/.vaultctl/session.json` (Contains encrypted session data - automatically managed)
- **Backups:** `~/.vaultctl/backups/vault-*.enc`

**Windows:**
- **Configuration:** `%USERPROFILE%\.vaultctl\config.json`
- **Vault file:** `%USERPROFILE%\.vaultctl\vault.db`
- **Session file:** `%USERPROFILE%\.vaultctl\session.json`
- **Backups:** `%USERPROFILE%\.vaultctl\backups\vault-*.enc`

The application automatically uses the correct path separators for your operating system.

## Advanced Usage

### Custom Table Name

If you want to use a different DynamoDB table:

1. Update `terraform.tfvars`:
   ```hcl
   dynamodb_table_name = "my_custom_table"
   ```

2. Apply Terraform changes:
   ```bash
   cd terraform
   terraform apply
   ```

3. Update `~/.vaultctl/config.json`:
   ```json
   {
     "table_name": "my_custom_table"
   }
   ```

### Multi-User Scenarios

For multiple users sharing the same DynamoDB table:

1. Set different `user_id` in config.json for each user:
   ```json
   {
     "user_id": "user1"
   }
   ```

2. Each user's vault will be stored separately in DynamoDB with:
   - PK: `USER#user1`
   - SK: `VAULT`

### Offline Mode

vaultctl works completely offline. DynamoDB is optional for:
- Multi-device sync
- Cloud backup
- Remote access

All operations work locally without DynamoDB.

### Session Management

vaultctl uses session-based unlocking for convenience:
- **Unlock once:** Enter master password when you start working
- **Work freely:** All commands work without password prompts for 30 minutes
- **Auto-expire:** Session expires after 30 minutes of inactivity
- **Manual lock:** Use `vaultctl lock` to end session early
- **Secure:** Session key stored encrypted on disk, protected by AWS Secrets Manager

- **Session file location:** `~/.vaultctl/session.json`
- **Session timeout:** 30 minutes (default)

## Command Reference

### Using Build Scripts

**macOS/Linux - run.sh:**

The `run.sh` script provides a convenient way to build and run vaultctl:

```bash
./run.sh build
# Build the application (checks prerequisites, downloads deps, compiles)

./run.sh run [command] [args...]
# Run vaultctl with the specified command and arguments
# Example: ./run.sh run init
# Example: ./run.sh run add --name github --username user@example.com

./run.sh build-and-run [command] [args...]
# Build the application and then run it with the specified command
# Example: ./run.sh build-and-run list

./run.sh clean
# Remove build artifacts (the vaultctl binary)

./run.sh help
# Show run.sh usage information

./run.sh
# If no command is specified, checks prerequisites, builds if needed,
# and shows vaultctl help
```

**Windows - run.bat or run.ps1:**

Windows users can use the equivalent batch or PowerShell scripts:

```cmd
REM Using run.bat (Command Prompt)
run.bat build
run.bat run init
run.bat run add --name github --username user@example.com
run.bat build-and-run list
run.bat clean
run.bat help
```

```powershell
# Using run.ps1 (PowerShell)
.\run.ps1 build
.\run.ps1 run init
.\run.ps1 run add --name github --username user@example.com
.\run.ps1 build-and-run list
.\run.ps1 clean
.\run.ps1 help
```

### Direct vaultctl Commands

Once built, you can run vaultctl directly:

```bash
vaultctl init
# Initialize a new vault

vaultctl unlock
# Unlock the vault with master password (creates a 30-minute session)

vaultctl lock
# Lock the vault and clear the session

vaultctl add [flags]
# Add a new password entry
# Flags: --name, --username, --url, --notes, --backup-codes, --no-sync

vaultctl get <name_or_id>
# Get a password entry by name or ID (displays all fields including backup codes)

vaultctl update <name_or_id> [flags]
# Update an existing entry
# Flags: --name, --username, --password, --url, --notes, --backup-codes, --no-sync

vaultctl list
# List all entries (without passwords)

vaultctl remove <name_or_id> [flags]
# Remove an entry by name or ID
# Flags: --no-sync

vaultctl sync
# Sync vault with DynamoDB

vaultctl backup [output_path]
# Create an encrypted backup

vaultctl restore [backup_path]
# Restore vault from a backup
# If no path provided, lists available backups for selection

vaultctl rotate-master
# Change the master password

vaultctl --help
# Show help for vaultctl

vaultctl [command] --help
# Show help for a specific command
```

## Architecture

### Key Derivation

1. Master password → Argon2id KDF → Master Key
2. Master Key encrypts Vault Key (random 32-byte key)
3. Vault Key encrypts all vault entries

### Storage

- **Local:** Encrypted vault stored at `~/.vaultctl/vault.db`
- **Remote:** Encrypted vault blob stored in DynamoDB table

### Vault Format

The vault is stored as an encrypted JSON structure containing:
- Salt for master key derivation
- Encrypted vault key
- KDF parameters
- Encrypted vault ciphertext
- Version and metadata

### Security Considerations

- **Master password:** Never logged, never stored, never sent to AWS
- **In-memory secrets:** Secrets are kept in memory only during the CLI session and zeroized after use
- **Encryption:** All data is encrypted with XChaCha20-Poly1305 using keys derived from your master password via Argon2id
- **Zero-knowledge:** DynamoDB never sees:
  - Master password
  - Master key
  - Vault key
  - Plaintext vault or entries
- **Session security:** Session master key stored in AWS Secrets Manager, session data encrypted on disk

## License

MIT

## Support

For issues, questions, or contributions:
- Check this README.md file
- Review the DESIGNDOC.md for architecture details
- Ensure all prerequisites are met
- Verify AWS configuration

**Remember:** This is a zero-knowledge password manager. If you lose your master password, your vault cannot be recovered. Always maintain backups!
