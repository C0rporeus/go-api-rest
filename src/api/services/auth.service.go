package services

import (
	userModel "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/apiresponse"
	"backend-yonathan/src/pkg/constants"
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

var (
	putItemFunc = func(dbClient *dynamodb.Client, input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
		return dbClient.PutItem(context.Background(), input)
	}
	getItemFunc = func(dbClient *dynamodb.Client, input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
		return dbClient.GetItem(context.Background(), input)
	}
	queryFunc = func(dbClient *dynamodb.Client, input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
		return dbClient.Query(context.Background(), input)
	}
)

func SaveUser(dbClient *dynamodb.Client, user userModel.User) error {
	if user.UserId == "" {
		user.UserId = uuid.New().String()
	}
	tableName := constants.TableName()
	input := &dynamodb.PutItemInput{
		Item: map[string]types.AttributeValue{
			"UserId":   &types.AttributeValueMemberS{Value: user.UserId},
			"email":    &types.AttributeValueMemberS{Value: user.Email},
			"password": &types.AttributeValueMemberS{Value: user.Password},
			"username": &types.AttributeValueMemberS{Value: user.UserName},
		},
		TableName: aws.String(tableName),
	}
	_, err := putItemFunc(dbClient, input)
	if err != nil {
		return err
	}
	return nil
}

// GetUserById looks up a user by the "UserId" partition key.
func GetUserById(dbClient *dynamodb.Client, id string) (userModel.User, error) {
	var user userModel.User
	tableName := constants.TableName()

	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"UserId": &types.AttributeValueMemberS{Value: id},
		},
	}

	result, err := getItemFunc(dbClient, input)
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
	tableName := constants.TableName()
	input := &dynamodb.QueryInput{
		TableName: aws.String(tableName),
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
	result, err := queryFunc(dbClient, input)

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

// Register godoc
// @Summary      Registro de usuarios
// @Description  Crea una cuenta nueva y devuelve un JWT
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        user  body  object{email=string,password=string,username=string}  true  "Datos de registro"
// @Success      200  {object}  map[string]string  "token"
// @Failure      400  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/register [post]
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

// Login godoc
// @Summary      Login de usuarios
// @Description  Autentica con email/password y devuelve un JWT
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        credentials  body  object{email=string,password=string}  true  "Credenciales"
// @Success      200  {object}  map[string]string  "token"
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/login [post]
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

// GetCurrentUser godoc
// @Summary      Usuario autenticado
// @Description  Devuelve userId y username del JWT actual
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      401  {object}  map[string]interface{}
// @Router       /api/private/me [get]
func GetCurrentUser(c *fiber.Ctx) error {
	return apiresponse.Success(c, fiber.Map{
		"userId":   c.Locals("userId"),
		"username": c.Locals("username"),
	})
}

// RefreshToken godoc
// @Summary      Renovar token JWT
// @Description  Genera un nuevo JWT a partir del token actual (debe ser valido)
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]string  "token"
// @Failure      401  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]interface{}
// @Router       /api/private/refresh [post]
func RefreshToken(c *fiber.Ctx) error {
	userId, _ := c.Locals("userId").(string)
	username, _ := c.Locals("username").(string)

	if userId == "" {
		return apiresponse.Error(c, fiber.StatusUnauthorized, "invalid_session", "Sesion invalida", nil)
	}

	token, err := jwtManager.GenerateToken(userId, username)
	if err != nil {
		return apiresponse.Error(c, fiber.StatusInternalServerError, "token_generation_failed", "No se pudo renovar el token", err.Error())
	}

	return apiresponse.Success(c, fiber.Map{"token": token})
}
