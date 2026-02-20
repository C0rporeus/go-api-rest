package dynamodbrepo

import (
	"context"
	"fmt"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// Injectable DynamoDB operation vars (swap in tests).
var (
	putItemFunc = func(client *dynamodb.Client, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
		return client.PutItem(context.Background(), input)
	}
	getItemFunc = func(client *dynamodb.Client, input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
		return client.GetItem(context.Background(), input)
	}
	queryFunc = func(client *dynamodb.Client, input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
		return client.Query(context.Background(), input)
	}
)

// UserRepository is the DynamoDB implementation of repository.UserRepository.
type UserRepository struct {
	client *dynamodb.Client
}

// NewUserRepository creates a new DynamoDB-backed UserRepository.
func NewUserRepository(client *dynamodb.Client) *UserRepository {
	return &UserRepository{client: client}
}

// SaveUser persists a user to DynamoDB. Generates a UserId if empty.
func (r *UserRepository) SaveUser(ctx context.Context, user models.User) error {
	if user.UserId == "" {
		user.UserId = uuid.New().String()
	}
	input := &dynamodb.PutItemInput{
		TableName: aws.String(constants.TableName()),
		Item: map[string]types.AttributeValue{
			"UserId":   &types.AttributeValueMemberS{Value: user.UserId},
			"email":    &types.AttributeValueMemberS{Value: user.Email},
			"password": &types.AttributeValueMemberS{Value: user.Password},
			"username": &types.AttributeValueMemberS{Value: user.UserName},
		},
	}
	_, err := putItemFunc(r.client, input)
	return err
}

// GetUserByID fetches a user by primary key.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (models.User, error) {
	var user models.User
	input := &dynamodb.GetItemInput{
		TableName: aws.String(constants.TableName()),
		Key: map[string]types.AttributeValue{
			"UserId": &types.AttributeValueMemberS{Value: id},
		},
	}
	result, err := getItemFunc(r.client, input)
	if err != nil {
		return user, err
	}
	if result.Item == nil {
		return user, fmt.Errorf("%w: user %s", repository.ErrNotFound, id)
	}
	err = attributevalue.UnmarshalMap(result.Item, &user)
	return user, err
}

// GetUserByEmail fetches a user by email via the GSI.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	input := &dynamodb.QueryInput{
		TableName: aws.String(constants.TableName()),
		IndexName: aws.String(constants.DynamoDBEmailIndex),
		KeyConditions: map[string]types.Condition{
			"email": {
				ComparisonOperator: types.ComparisonOperatorEq,
				AttributeValueList: []types.AttributeValue{
					&types.AttributeValueMemberS{Value: email},
				},
			},
		},
	}
	result, err := queryFunc(r.client, input)
	if err != nil {
		return user, err
	}
	if len(result.Items) == 0 {
		return user, fmt.Errorf("%w: email %s", repository.ErrNotFound, email)
	}
	err = attributevalue.UnmarshalMap(result.Items[0], &user)
	return user, err
}
