package authServices

import (
	userModel "backend-yonathan/src/models"
	jwtManager "backend-yonathan/src/pkg/utils"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func SaveUser(dbClient *dynamodb.Client, user userModel.User) error {
	if user.UserId == "" {
		user.UserId = uuid.New().String()
	}
	input := &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"UserId":   &types.AttributeValueMemberS{Value: user.UserId},
			"email":    &types.AttributeValueMemberS{Value: user.Email},
			"password": &types.AttributeValueMemberS{Value: user.Password},
		},
		TableName: aws.String("users"),
	}
	_, err := dbClient.PutItem(context.Background(), input)
	if err != nil {
		return err
	}
	return nil
}

func GetUserById(dbClient *dynamodb.Client, id string) (userModel.User, error) {
	var user userModel.User
	input := &dynamodb.GetItemInput{
		TableName: aws.String("users"),
		Key: map[string]types.AttributeValue{
			"userId": &types.AttributeValueMemberS{Value: id},
		},
	}

	result, err := dbClient.GetItem(context.Background(), input)
	if err != nil {
		return user, err
	}

	if result.Item == nil {
		return user, fmt.Errorf("user not found")
	}

	err = attributevalue.UnmarshalMap(result.Item, &user)
	if err != nil {
		return user, err
	}

	return user, nil
}

func GetUserByEmail(dbClient *dynamodb.Client, email string) (userModel.User, error) {
	var user userModel.User
	input := &dynamodb.QueryInput{
		TableName: aws.String("users"),
		IndexName: aws.String("email-index"),
		KeyConditions: map[string]types.Condition{
			"email": {
				ComparisonOperator: types.ComparisonOperatorEq,
				AttributeValueList: []types.AttributeValue{
					&types.AttributeValueMemberS{Value: email},
				},
			},
		},
	}
	result, err := dbClient.Query(context.Background(), input)

	if err != nil {
		return user, err
	}

	if len(result.Items) == 0 {
		return user, fmt.Errorf("user not found")
	}

	err = attributevalue.UnmarshalMap(result.Items[0], &user)
	if err != nil {
		return user, err
	}
	return user, nil
}

func Register(c *fiber.Ctx, dbClient *dynamodb.Client) error {
	var user userModel.User
	if err := c.BodyParser(&user); err != nil {
		return err
	}

	_, err := GetUserByEmail(dbClient, user.Email)
	if err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "El usuario ya existe"})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	err = SaveUser(dbClient, user)
	if err != nil {
		return err
	}
	token, err := jwtManager.GenerateToken(user.UserId)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"token": token})
}

func Login(c *fiber.Ctx, dbClient *dynamodb.Client) error {
	var loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&loginRequest); err != nil {
		return err
	}

	user, err := GetUserByEmail(dbClient, loginRequest.Email)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password))
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Unauthorized"})
	}

	token, err := jwtManager.GenerateToken(user.UserId)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"token": token})
}
