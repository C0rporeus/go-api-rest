# API REST — Portfolio Personal

Backend del sitio web personal orientado a publicar información personal y técnica para profesionales en tecnología.

## Stack

- [Go](https://golang.org/) 1.25
- [Go Fiber](https://gofiber.io/) v3 — Framework web de alto rendimiento
- [JWT](https://jwt.io/) — Autenticación y autorización
- [AWS DynamoDB](https://aws.amazon.com/es/dynamodb/) — Base de datos NoSQL
- [Swagger (swag)](https://github.com/swaggo/swag) — Documentación de API

## Variables de entorno

Copiar `.env.example` y ajustar valores. Las credenciales AWS se inyectan via entorno o `AWS_PROFILE`:

```bash
PORT=3100
JWT_SECRET=change-me
JWT_EXPIRY_HOURS=24
AWS_REGION=us-east-1
DYNAMO_DB_TABLE=users
PORTFOLIO_DATA_DIR=./data
CORS_ALLOWED_ORIGINS=http://localhost:3000
```

Para desarrollo local con credenciales AWS compartidas:

```bash
AWS_PROFILE=your-profile
```

Variables opcionales de observabilidad:

```bash
OPS_ALERT_MIN_REQUESTS=20
OPS_WARN_5XX_RATE=0.05
OPS_CRITICAL_5XX_RATE=0.10
OPS_WARN_AUTH_FAIL_RATE=0.10
OPS_CRITICAL_AUTH_FAIL_RATE=0.20
OPS_WINDOW_SECONDS=300
OPS_HEALTH_HISTORY_LIMIT=100
OPS_SUMMARY_SAMPLE_SIZE=50
```

## Iniciar

```bash
# Con hot-reload (requiere air)
air

# Directo
go run main.go
```

El servidor inicia en `http://localhost:3100`.

## Endpoints (28 totales)

### Públicos (5)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/login` | Autenticación con email/password |
| POST | `/api/register` | Registro de usuario |
| POST | `/api/contact` | Formulario de contacto |
| GET | `/api/experiences` | Listar experiencias públicas |
| GET | `/api/skills` | Listar skills públicas |

### Tools (8, públicos)

| Método | Ruta | Descripción |
|--------|------|-------------|
| POST | `/api/tools/base64/encode` | Codificar Base64 |
| POST | `/api/tools/base64/decode` | Decodificar Base64 |
| GET | `/api/tools/uuid/v4` | Generar UUID v4 |
| POST | `/api/tools/certs/self-signed` | Generar certificado autofirmado |
| GET | `/api/tools/dns/resolve` | Resolución DNS |
| GET | `/api/tools/dns/propagation` | Propagación DNS por tipo de registro |
| GET | `/api/tools/dns/mail-records` | Registros MX, SPF, DKIM, DMARC |
| GET | `/api/tools/dns/blacklist` | Verificación DNSBL (6 proveedores) |

### Privados (15, requieren JWT)

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/private/me` | Usuario autenticado |
| POST | `/api/private/refresh` | Renovar JWT |
| GET | `/api/private/experiences` | Listar todas las experiencias |
| POST | `/api/private/experiences` | Crear experiencia |
| PUT | `/api/private/experiences/:id` | Actualizar experiencia |
| DELETE | `/api/private/experiences/:id` | Eliminar experiencia |
| GET | `/api/private/skills` | Listar todas las skills |
| POST | `/api/private/skills` | Crear skill |
| PUT | `/api/private/skills/:id` | Actualizar skill |
| DELETE | `/api/private/skills/:id` | Eliminar skill |
| GET | `/api/private/ops/metrics` | Métricas operativas |
| GET | `/api/private/ops/alerts` | Alertas operativas |
| GET | `/api/private/ops/health` | Estado de salud |
| GET | `/api/private/ops/history` | Historial de estados |
| GET | `/api/private/ops/summary` | Resumen para semáforo |

### Documentación

| Ruta | Descripción |
|------|-------------|
| `/swagger/*` | Swagger UI |

## Documentación de API

Para regenerar la documentación Swagger:

```bash
swag init
```

## Testing

```bash
# Ejecutar todos los tests
go test ./...

# Verificar cobertura (gate >= 80%)
bash scripts/check_coverage.sh
```

## Lint

```bash
golangci-lint run
```
