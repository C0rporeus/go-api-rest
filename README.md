# API Rest - Fiver

## Backend básico, sitio web personal

Se trata de aplicación que resolverá sitio web personal orientado a publicar información personal y técnica para quien pueda interesar en temas tecnológicos en las áreas y temas de intereses para ingenieros en diferentes ámbitos e intereses.

Esta usa entre sus tecnologias:

<!-- link url -->

[Go Fiber]: https://gofiber.io/
[Go]: https://golang.org/
[JWT]: https://jwt.io/
[AWS DynamoDB]: https://aws.amazon.com/es/dynamodb/
[AWS S3]: https://aws.amazon.com/es/s3/

- [Go Fiber] - Framework web para [Go]
- [Go] - Lenguaje de programación
- [JWT] - Autenticación y autorización
- [AWS DynamoDB] - Base de datos no relacional
- [AWS S3] - Almacenamiento de archivos

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

Puede inicializar esta app con el comando `air` o `go run main.go`

```bash
air
```

```bash
go run main.go
```

