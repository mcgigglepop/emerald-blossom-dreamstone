package secrets

import (
	"context"
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

// GetSessionKey retrieves the session master key from Secrets Manager
// The secret must exist beforehand - it will not be created automatically
func (smc *SecretsManagerClient) GetSessionKey(ctx context.Context) ([]byte, error) {
	// Get the secret from Secrets Manager
	result, err := smc.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(smc.secretName),
	})
	if err != nil {
		// Check if it's a ResourceNotFoundException
		if _, ok := err.(*types.ResourceNotFoundException); ok {
			return nil, fmt.Errorf("secret '%s' not found in AWS Secrets Manager. Please create it first", smc.secretName)
		}
		// Check error code as fallback
		if err, ok := err.(interface{ ErrorCode() string }); ok && err.ErrorCode() == "ResourceNotFoundException" {
			return nil, fmt.Errorf("secret '%s' not found in AWS Secrets Manager. Please create it first", smc.secretName)
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

