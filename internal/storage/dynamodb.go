package storage

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBStorage handles DynamoDB operations
type DynamoDBStorage struct {
	client    *dynamodb.Client
	tableName string
	userID    string
}

// DynamoDBItem represents the item structure in DynamoDB
type DynamoDBItem struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	VaultID   string `dynamodbav:"vault_id"`
	VaultBlob string `dynamodbav:"vault_blob"` // JSON string of EncryptedVault
	Version   int64  `dynamodbav:"version"`
	ModifiedAt string `dynamodbav:"modified_at"`
	DeviceID  string `dynamodbav:"device_id"`
}

// NewDynamoDBStorage creates a new DynamoDB storage instance
func NewDynamoDBStorage(tableName, userID string) (*DynamoDBStorage, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &DynamoDBStorage{
		client:    dynamodb.NewFromConfig(cfg),
		tableName: tableName,
		userID:    userID,
	}, nil
}

// GetDeviceID returns a unique device identifier
func GetDeviceID() string {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s-%d", hostname, os.Getpid())
}

// SaveVault saves an encrypted vault to DynamoDB
func (ds *DynamoDBStorage) SaveVault(ctx context.Context, ev *EncryptedVault, expectedVersion int64) error {
	vaultBlob, err := ev.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize vault: %w", err)
	}

	item := DynamoDBItem{
		PK:        fmt.Sprintf("USER#%s", ds.userID),
		SK:        "VAULT",
		VaultID:   ev.VaultID,
		VaultBlob: string(vaultBlob),
		Version:   ev.Version,
		ModifiedAt: ev.ModifiedAt,
		DeviceID:  GetDeviceID(),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("failed to marshal item: %w", err)
	}

	// Conditional write to prevent overwriting newer versions
	conditionExpr := "attribute_not_exists(version) OR version = :expectedVersion"
	exprAttrValues := map[string]types.AttributeValue{
		":expectedVersion": &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", expectedVersion)},
	}

	input := &dynamodb.PutItemInput{
		TableName:                 aws.String(ds.tableName),
		Item:                      av,
		ConditionExpression:       aws.String(conditionExpr),
		ExpressionAttributeValues: exprAttrValues,
	}

	_, err = ds.client.PutItem(ctx, input)
	if err != nil {
		var condCheckErr *types.ConditionalCheckFailedException
		if errors.As(err, &condCheckErr) {
			return fmt.Errorf("version conflict: remote vault has been updated. Run 'vaultctl sync' first")
		}
		return fmt.Errorf("failed to save vault: %w", err)
	}

	return nil
}

// LoadVault loads an encrypted vault from DynamoDB
func (ds *DynamoDBStorage) LoadVault(ctx context.Context) (*EncryptedVault, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(ds.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", ds.userID)},
			"SK": &types.AttributeValueMemberS{Value: "VAULT"},
		},
	}

	result, err := ds.client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault from DynamoDB: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("vault not found in DynamoDB")
	}

	var item DynamoDBItem
	if err := attributevalue.UnmarshalMap(result.Item, &item); err != nil {
		return nil, fmt.Errorf("failed to unmarshal item: %w", err)
	}

	ev, err := EncryptedVaultFromJSON([]byte(item.VaultBlob))
	if err != nil {
		return nil, fmt.Errorf("failed to parse vault blob: %w", err)
	}

	return ev, nil
}

// SyncVault handles syncing between local and remote vaults
func (ds *DynamoDBStorage) SyncVault(ctx context.Context, localEV *EncryptedVault) (*EncryptedVault, error) {
	remoteEV, err := ds.LoadVault(ctx)
	if err != nil {
		// If remote doesn't exist, push local
		if err.Error() == "vault not found in DynamoDB" {
			return localEV, ds.SaveVault(ctx, localEV, localEV.Version-1)
		}
		return nil, err
	}

	// If local is newer or same, push local
	if localEV.Version >= remoteEV.Version {
		if err := ds.SaveVault(ctx, localEV, remoteEV.Version); err != nil {
			return nil, err
		}
		return localEV, nil
	}

	// Remote is newer, return remote
	return remoteEV, nil
}

