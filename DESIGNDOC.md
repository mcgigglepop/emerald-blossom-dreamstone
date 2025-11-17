Here’s a design doc you can use as a starting point for your Go + Cobra + AWS/DynamoDB CLI password manager, with **all crypto client-side** and **zero-knowledge** from the server’s point of view.

---

## 1. Overview

**Working name:** `vaultctl` (change as you like)
**Type:** CLI password manager
**Stack:**

* **Language:** Go
* **CLI:** Cobra
* **Storage (cloud):** DynamoDB
* **Local storage:** Encrypted file (for cache + offline use)
* **Crypto:** All encryption/decryption client-side. Server (DynamoDB) stores only ciphertext and metadata.

Core idea:

* User has a **master password**.
* Master password is used only locally to derive encryption keys.
* DynamoDB just stores encrypted blobs and opaque metadata (timestamps, IDs, etc.).
* No plaintext secrets, no master password, and no key material are ever sent to AWS.

---

## 2. Goals & Non-Goals

### Goals

* **Zero-knowledge / client-side crypto**

  * Master password never leaves client.
  * Server never sees decrypted vault or entries.
  * All encryption, decryption, and key derivation happens locally.
* **Multi-device sync**

  * Vault can be synced across devices via DynamoDB.
* **Offline-friendly**

  * Vault cached locally in an encrypted file.
* **Simple CLI UX**

  * Intuitive commands with Cobra (`init`, `unlock`, `add`, `get`, `list`, `remove`, `sync`, `backup`, etc.).
* **Strong cryptography**

  * Use modern, well-reviewed primitives (e.g., Argon2id, AES-GCM or XChaCha20-Poly1305).

### Non-Goals (initial version)

* No GUI or browser plugin (CLI only).
* No multi-user sharing of passwords (e.g., team vaults) — at least not in v1.
* No offline brute-force protections beyond KDF tuning (no remote rate limiting).

---

## 3. High-Level Architecture

### Components

1. **CLI (Go + Cobra)**

   * Entry point for all operations.
   * Reads configuration (AWS region, table name, etc.).
   * Handles user prompts for the master password and entry fields.

2. **Crypto Module (Go package)**

   * Responsible for:

     * Key derivation from master password (KDF).
     * Vault key management.
     * Encrypting/decrypting vault & entries.
   * Exposes simple functions like `EncryptVault`, `DecryptVault`, `EncryptEntry`, `DecryptEntry`.

3. **Storage Module**

   * **Local storage** (e.g., `$HOME/.vaultctl/vault.db`):

     * Encrypted vault blob cached locally for quick access and offline use.
   * **Remote storage (DynamoDB)**:

     * Stores encrypted vault state and/or per-entry encrypted objects.
     * Also stores metadata such as version, modified timestamp, device IDs, etc.

4. **AWS Integration**

   * Uses AWS SDK for Go to:

     * Put/get/update vault items in DynamoDB.
   * Authentication to AWS via:

     * Standard AWS credential chain (env vars, profiles, instance roles, etc.).
   * No secret data in Dynamo beyond ciphertext.

---

## 4. Crypto Design

### 4.1 Keys & Passwords

Terminology:

* **Master Password**: Chosen by user, typed into CLI.
* **Master Key (MK)**: Derived from master password via KDF (Argon2id).
* **Vault Key (VK)**: Random symmetric key used to encrypt the actual vault contents.
* **Entry Keys** (optional): Either:

  * Use VK directly for all entries, or
  * Derive per-entry keys from VK and entry ID (HKDF) for additional compartmentalization.

Recommended structure:

1. On **vault init**:

   * Generate:

     * `salt_master` (random 16–32 bytes).
     * `vault_key` (VK), random 32 bytes.
   * Derive `master_key = KDF(master_password, salt_master)`.
   * Encrypt `vault_key` using `master_key`.
   * Store:

     * `salt_master`
     * `enc_vault_key`
   * All **encrypted and stored locally**, plus pushed to DynamoDB as part of vault metadata.

2. On **vault unlock**:

   * Read `salt_master` and `enc_vault_key`.
   * Derive `master_key` from provided master password.
   * Decrypt `vault_key`.
   * Keep `vault_key` in memory (never written to disk or AWS).

### 4.2 Algorithms

* **KDF:** Argon2id with configurable parameters:

  * `memory` (e.g., 64–256 MB),
  * `iterations` (e.g., 2–4),
  * `parallelism` (e.g., 1–4).
* **AEAD cipher:** one of:

  * AES-256-GCM (if using stdlib + crypto libraries), or
  * XChaCha20-Poly1305 (for better nonce safety).
* **Randomness:** `crypto/rand`.

### 4.3 Vault Format

Vault logical structure (plaintext model):

```json
{
  "schema_version": 1,
  "vault_id": "uuid",
  "entries": [
    {
      "id": "uuid",
      "name": "github-personal",
      "username": "user@example.com",
      "password": "supersecret",
      "url": "https://github.com/login",
      "notes": "2FA enabled via TOTP",
      "created_at": "2025-11-16T22:00:00Z",
      "updated_at": "2025-11-16T22:00:00Z"
    },
    ...
  ]
}
```

Stored format (encrypted):

```json
{
  "schema_version": 1,
  "vault_id": "uuid",
  "salt_master": "<base64>",
  "enc_vault_key": "<base64>",
  "kdf_params": {
    "algo": "argon2id",
    "memory": 65536,
    "iterations": 3,
    "parallelism": 1
  },
  "cipher": "xchacha20poly1305",
  "ciphertext": "<base64>",  // encrypted plaintext vault JSON
  "nonce": "<base64>",
  "modified_at": "2025-11-16T22:00:00Z",
  "version": 42
}
```

**Important:** DynamoDB only sees the **outer JSON object**. `ciphertext` is opaque.

---

## 5. Data Model (DynamoDB)

Depending on design, you can store **one vault blob per user** or **one item per entry**. For simplicity and strong consistency, v1 will use **one vault per user**.

### 5.1 Table Schema

**Table name:** `vaultctl_vaults`

**Primary key:**

* `PK` (partition key): `USER#<user_id>` (if multi-user) or just `VAULT#<vault_id>` if single-user.
* `SK` (sort key): constant like `VAULT` (or version if you want historical versions).

Example items (single current version item):

```json
{
  "PK": "USER#12345",
  "SK": "VAULT",
  "vault_id": "uuid",
  "vault_blob": "<encrypted vault JSON as string>",
  "version": 42,  // monotonically increasing
  "modified_at": "2025-11-16T22:00:00Z",
  "device_id": "machine-abc"
}
```

If you want historical versions:

* `SK`: `VAULT#<version>`
* Use a GSI to query latest version or store a small pointer item for “current version”.

### 5.2 User Identity

v1 options:

* **Simplest:** assume single AWS user (your own AWS account). `user_id` is a static value in config.
* **Multi-user:** integrate with an auth layer (e.g., Cognito hosted UI + tokens). But **Cognito never sees master password**, it only handles login to your backend. The CLI could take a user token to identify which vault in Dynamo to read/write.

For now, design for **single user, single AWS account** and leave multi-user as future work.

---

## 6. CLI Design (Cobra Commands)

Top-level command: `vaultctl`

### 6.1 Commands

1. `vaultctl init`

   * Prompts:

     * Master password (+ confirmation).
   * Actions:

     * Generate `salt_master`, `vault_key`, initial empty vault.
     * Derive `master_key` from password.
     * Encrypt vault and `vault_key`.
     * Write to local file.
     * Push initial vault to DynamoDB.

2. `vaultctl unlock`

   * Prompts:

     * Master password.
   * Actions:

     * Load encrypted vault from local file (or Dynamo if local missing).
     * Derive `master_key`, decrypt `vault_key`.
     * Decrypt vault into memory.
     * Keep unlocked vault in memory while CLI process runs (for interactive sessions).

3. `vaultctl add`

   * Flags: `--name`, `--username`, `--url`, `--notes`.
   * Prompts for password input (hidden).
   * Requires unlocked vault (or does `unlock` flow inline).
   * Adds entry, updates vault in memory.
   * Re-encrypts vault, writes to disk, and optionally syncs to Dynamo (or `--no-sync` flag).

4. `vaultctl get <name_or_id>`

   * Resolve entry by name or ID.
   * Requires unlocked vault.
   * Options:

     * Print to stdout (warn user).
     * Optional integration with OS clipboard (if you want, though that’s OS-specific).

5. `vaultctl list`

   * Lists entry names/IDs and some metadata (no secrets).
   * Requires unlocked vault.

6. `vaultctl remove <name_or_id>`

   * Deletes entry from vault, re-encrypts, syncs.

7. `vaultctl sync`

   * Pull latest vault from DynamoDB (with version checks).
   * Merge local changes or prompt user if conflicts (v2+).
   * Push updated vault.

8. `vaultctl backup`

   * Creates a **local** encrypted backup file (e.g., `vault-2025-11-16T22-00-00Z.enc`).
   * Optionally uploads backups to S3 (also encrypted).

9. `vaultctl rotate-master`

   * Allows changing master password.
   * Flow:

     * Require current master password (to get current `vault_key`).
     * Generate new `salt_master`.
     * Derive new `master_key` from new password.
     * Re-encrypt `vault_key` with new `master_key`.
     * Update `salt_master`, `enc_vault_key`, and write back to local + Dynamo.

---

## 7. AWS Integration Details

### 7.1 DynamoDB Operations

* **Get vault**

  * `GetItem` on `PK`, `SK = "VAULT"`.
* **Put vault**

  * `PutItem` with conditional expression on `version` to prevent overwriting a newer version:

    * `ConditionExpression: "attribute_not_exists(version) OR version = :expectedVersion"`.
* **Conflict handling**

  * If conditional write fails:

    * Fetch remote version.
    * Compare with local.
    * Either:

      * Prompt user to choose which to keep.
      * Or implement a merge (future enhancement).

### 7.2 IAM Permissions

Minimum IAM policy attached to the AWS identity used by the CLI:

* `dynamodb:GetItem`
* `dynamodb:PutItem`
* (optional) `dynamodb:UpdateItem`, `Query` if you do versions/history.

No permissions for KMS are needed in v1 because **all crypto is client-side**.

If you later want server-side encrypted backups (S3 + KMS), that’s an additive change.

---

## 8. Key Flows

### 8.1 Initialization Flow

1. User runs `vaultctl init`.
2. CLI:

   * Prompts for master password twice.
   * Generates `salt_master` and `vault_key`.
   * Derives `master_key` via Argon2id.
   * Encrypts initial empty vault JSON and `vault_key`.
   * Writes encrypted vault file locally.
   * Calls DynamoDB `PutItem` to store encrypted vault.

### 8.2 Unlock Flow

1. User runs `vaultctl unlock` (or `add/get/list` which implicitly unlock).
2. CLI:

   * Reads local encrypted vault blob.
   * Reads `salt_master`, `enc_vault_key`, `ciphertext`, etc.
   * Prompts for master password.
   * Derives `master_key`.
   * Decrypts `vault_key` and vault.
   * Keeps decrypted vault only in memory for this process.

### 8.3 Add Entry Flow

1. User runs `vaultctl add --name github`.
2. CLI:

   * Ensures vault is unlocked (may prompt).
   * Adds new entry to in-memory vault.
   * Re-encrypts vault with `vault_key`.
   * Updates local file.
   * Increments `version` and `modified_at`.
   * Writes to DynamoDB with conditional `version` check.

---

## 9. Security Considerations

* **Master password**

  * Never logged, never stored.
  * Avoid sending it to child processes or shell.
* **In-memory secrets**

  * Keep secrets in memory only as long as needed.
  * Avoid unnecessary copies (be mindful of `string` handling in Go).
  * Consider custom types backed by `[]byte` that can be zeroized.
* **Logging**

  * Disable logging of sensitive values.
  * Distinguish between info logs and debug logs (no secrets in any logs).
* **Temporary files**

  * Never write plaintext secrets to disk (including temp files).
* **Backup files**

  * Must always be encrypted with the same vault key or a separate backup key derived from master password.
* **Brute-force resistance**

  * Argon2id with strong parameters.
  * Encourage users to pick long master phrases.
* **No server-side key material**

  * DynamoDB never sees:

    * Master password
    * Master key
    * Vault key
    * Plaintext vault or entries

---

## 10. Future Enhancements

* Multi-user support with Cognito or another auth system.
* Team/shared vaults with asymmetric crypto (per-user public keys).
* Device-specific keys; possibly hardware-backed keys.
* Secure clipboard integration per OS.
* More advanced conflict resolution in sync.
* S3-based encrypted backups with KMS (still double-encrypted).

---

