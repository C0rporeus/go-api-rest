package authServices

import (
	"backend-yonathan/src/pkg/apiresponse"
	userModel "backend-yonathan/src/models"
	jwtManager "backend-yonathan/src/pkg/utils"
	"context"
	"fmt"
	"os"

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
			"username": &types.AttributeValueMemberS{Value: user.UserName},
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
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	_, err := GetUserByEmail(dbClient, user.Email)
	if err == nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "user_already_exists", "El usuario ya existe", nil)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "password_hash_failed", "No se pudo procesar la contrasena", err.Error())
	}
	user.Password = string(hashedPassword)
	err = SaveUser(dbClient, user)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "save_user_failed", "No se pudo registrar el usuario", err.Error())
	}
	token, err := jwtManager.GenerateToken(user.UserId, user.UserName)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "token_generation_failed", "No se pudo generar el token", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{"token": token})
}

func Login(c *fiber.Ctx, dbClient *dynamodb.Client) error {
	var loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&loginRequest); err != nil {
		return apiresponse.Error(c, fiber.StatusBadRequest, "invalid_payload", "Payload invalido", err.Error())
	}

	user, err := GetUserByEmail(dbClient, loginRequest.Email)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_credentials", "Unauthorized", nil)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password))
	if err != nil {
		return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_credentials", "Unauthorized", nil)
	}

	token, err := jwtManager.GenerateToken(user.UserId, user.UserName)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "token_generation_failed", "No se pudo generar el token", err.Error())
	}
	return apiresponse.Success(c, fiber.Map{"token": token})
}
