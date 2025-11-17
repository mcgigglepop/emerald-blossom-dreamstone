package secrets

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

// SecretsManagerClient wraps AWS Secrets Manager operations
type SecretsManagerClient struct {
	client    *secretsmanager.Client
	secretName string
	region    string
}

// NewSecretsManagerClient creates a new Secrets Manager client
func NewSecretsManagerClient(secretName, region string) (*SecretsManagerClient, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &SecretsManagerClient{
		client:     secretsmanager.NewFromConfig(cfg),
		secretName: secretName,
		region:     region,
	}, nil
}

// GetOrCreateSessionKey retrieves the session master key from Secrets Manager,
// or creates it if it doesn't exist
func (smc *SecretsManagerClient) GetOrCreateSessionKey(ctx context.Context) ([]byte, error) {
	// Try to get existing secret
	result, err := smc.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(smc.secretName),
	})
	if err != nil {
		// Check if it's a ResourceNotFoundException
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			// Secret doesn't exist, create it
			return smc.createSessionKey(ctx)
		}
		// Check error code as fallback
		if err, ok := err.(interface{ ErrorCode() string }); ok && err.ErrorCode() == "ResourceNotFoundException" {
			return smc.createSessionKey(ctx)
		}
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	// Decode the secret value (stored as base64)
	key, err := base64.StdEncoding.DecodeString(*result.SecretString)
	if err != nil {
		return nil, fmt.Errorf("failed to decode secret: %w", err)
	}

	return key, nil
}

// createSessionKey creates a new session master key and stores it in Secrets Manager
func (smc *SecretsManagerClient) createSessionKey(ctx context.Context) ([]byte, error) {
	// Generate a random 32-byte key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate session key: %w", err)
	}

	// Encode as base64 for storage
	encoded := base64.StdEncoding.EncodeToString(key)

	// Create the secret in Secrets Manager
	_, err := smc.client.CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(smc.secretName),
		SecretString: aws.String(encoded),
		Description:  aws.String("Vaultctl session master key for encrypting session keys"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}

	return key, nil
}

// IsAvailable checks if Secrets Manager is available
func (smc *SecretsManagerClient) IsAvailable(ctx context.Context) bool {
	_, err := smc.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(smc.secretName),
	})
	if err != nil {
		// Check if it's a ResourceNotFoundException (secret doesn't exist, but SM is available)
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			return true
		}
		// Check error code as fallback
		if err, ok := err.(interface{ ErrorCode() string }); ok {
			code := err.ErrorCode()
			if code == "ResourceNotFoundException" {
				return true
			}
		}
		// Other errors might indicate Secrets Manager is not available
		return false
	}
	return true
}

