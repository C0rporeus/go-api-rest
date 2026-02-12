# API Rest - Fiver

## Backend básico, sitio web personal

Se trata de aplicación que resolverá sitio web personal orientado a publicar información personal y técnica para quien pueda interesar en temas tecnológicos en las áreas y temas de intereses para ingenieros en diferentes ámbitos.

Esta usa entre sus tecnologias:

<!-- link url -->

[Go Fiber]: https://gofiber.io/
[Go]: https://golang.org/
[JWT]: https://jwt.io/
[AWS DynamoDB]: https://aws.amazon.com/es/dynamodb/
[AWS S3]: https://aws.amazon.com/es/s3/
[swag init]: https://github.com/swaggo/swag

- [Go Fiber] - Framework web para [Go]
- [Go] - Lenguaje de programación
- [JWT] - Autenticación y autorización
- [AWS DynamoDB] - Base de datos no relacional
- [AWS S3] - Almacenamiento de archivos
- [swag init] - Documentación de API

incluir variables de entorno

```bash
export PORT=3000
export JWT_SECRET=secret
export DYNAMO_DB_REGION=us-east-1
export DYNAMO_DB_ENDPOINT=http://localhost:8000
export DYNAMO_DB_TABLE=users
export AWS_ACCESS_KEY_ID=accesskey
export AWS_SECRET_ACCESS_KEY=secretkey
export AWS_REGION=us-east-1
export AWS_S3_BUCKET=bucket
```

Endpoints principales:

- `POST /api/login`
- `POST /api/register`
- `GET /api/tools/uuid/v4`
- `POST /api/tools/base64/encode`
- `POST /api/tools/base64/decode`
- `POST /api/tools/certs/self-signed`
- `GET /api/private/me` (requiere `Authorization: Bearer <token>`)
- `GET /api/experiences` (contenido publico del portafolio)
- `GET /api/private/experiences` (admin)
- `POST /api/private/experiences` (admin)
- `PUT /api/private/experiences/:id` (admin)
- `DELETE /api/private/experiences/:id` (admin)

Puede inicializar esta app con el comando `air` o `go run main.go`

```bash
air
```

```bash
go run main.go
```

## Documentación de API

Para generar la documentación de la API, se usa [swag init]

```bash
swag init
```
El esquema de datos se debe dar a nivel de el router, en el archivo `docs/swagger.json`

```
// @Summary Login de usuarios
// @Description Login de usuarios api Yonathan Gutierrez Dev
// @Tags Auth
// @Accept json
// @Produce json
// @Param user body userModel.User true "User"
// @Success 200 {object} userModel.User
// @Failure 400 {string} string "bad request"
// @Router /api/login [post]
```

Igualmente el modelo de datos se debe dar a nivel de el modelo, en el archivo `docs/swagger.json`

```
// User model
// @Description Modelo de usuario.
type User struct {
    ID        string `json:"id"`
    Email     string `json:"email"`
    Password  string `json:"password"`
    FirstName string `json:"firstName"`
    LastName  string `json:"lastName"`
    Role      string `json:"role"`
}
```
