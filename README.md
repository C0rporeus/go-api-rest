# API REST — Portfolio Personal

Backend del sitio web personal orientado a publicar información personal y técnica para profesionales en tecnología.

## Stack base

- [Go](https://golang.org/) 1.25
- [Go Fiber](https://gofiber.io/) v3 — Framework web de alto rendimiento
- [JWT](https://jwt.io/) — Autenticación y autorización
- [Docker](https://www.docker.com/) — Empaquetado del backend para despliegue
- [Swagger (swag)](https://github.com/swaggo/swag) — Documentación de API

## Stack actual en GCP

Servicios confirmados en el proyecto `porfolio-58ea0`:

| Capa | Servicio GCP | Uso actual |
|------|--------------|-----------|
| Frontend hosting | **Firebase Hosting** | Publica el frontend exportado de Next.js desde `front-end/out` en `https://porfolio-58ea0.web.app` |
| Frontend proyecto | **Firebase Project** | Proyecto base compartido entre frontend y backend (`porfolio-58ea0`) |
| Backend hosting | **Cloud Run** | Ejecuta la API Go/Fiber como servicio contenedorizado (`porfolio-api`) |
| Contenedores | **Artifact Registry** | Almacena la imagen Docker del backend: `us-central1-docker.pkg.dev/porfolio-58ea0/porfolio/api:latest` |
| Base de datos | **Cloud Firestore (Native mode)** | Persistencia principal de usuarios, experiencias y skills en producción |
| Secretos | **Secret Manager** | Guarda secretos consumidos por Cloud Run, incluyendo `porfolio-jwt-secret` y `porfolio-admin-password` |
| Identidad de runtime | **IAM Service Account** | El servicio Cloud Run usa `porfolio-cr-sa@porfolio-58ea0.iam.gserviceaccount.com` |
| Dominio/API | **Cloud Run + dominio personalizado** | La API se expone vía Cloud Run y responde en `https://api.yonathangutierrez.dev` |

> Nota: el código implementa una capa de repositorios agnóstica (`DB_PROVIDER`), pero el despliegue real documentado aquí usa **exclusivamente servicios de GCP**.

### Qué usa cada parte

- **Frontend (`front-end`)**
  - Next.js exportado como sitio estático
  - Firebase Hosting para servir HTML, JS, CSS y cabeceras de cache
  - Consumo de API pública/privada vía `NEXT_PUBLIC_API_URL=https://api.yonathangutierrez.dev`

- **Backend (`go-api-rest`)**
  - API Go/Fiber ejecutada en Cloud Run
  - Imagen construida en Docker y publicada en Artifact Registry
  - Firestore como proveedor real de datos en producción (`DB_PROVIDER=firestore`)
  - Secret Manager para JWT y credenciales del usuario administrador
  - CORS configurado para `porfolio-58ea0.web.app`, `porfolio-58ea0.firebaseapp.com`, `www.yonathangutierrez.dev`, `yonathangutierrez.dev` y `localhost:3000`

## Variables de entorno

Copiar `.env.example` y ajustar valores. Para el entorno desplegado en GCP, estas son las variables relevantes:

```bash
PORT=3100
JWT_SECRET=change-me
JWT_EXPIRY_HOURS=24
DB_PROVIDER=firestore
GCP_PROJECT_ID=porfolio-58ea0
PORTFOLIO_DATA_DIR=./data
CORS_ALLOWED_ORIGINS=http://localhost:3000
GCS_BUCKET_NAME=porfolio-58ea0.appspot.com
```

`GCS_BUCKET_NAME`: bucket de Google Cloud Storage (o Firebase Storage) donde el backend sube las imágenes del admin/editor. Si no se define, `POST /api/private/upload-image` responde 503. Credenciales vía `GOOGLE_APPLICATION_CREDENTIALS` o ADC.

`SIGNED_URL_EXPIRY_HOURS` (opcional, default 168 = 7 días): duración de las URLs firmadas que el backend genera al servir imágenes. Las imágenes subidas al bucket son privadas; los endpoints de lectura (experiencias, skills) devuelven URLs firmadas V4 con expiración temporal.

Las alternativas de persistencia existen en el código, pero no forman parte del stack productivo actual y por eso no se listan aquí como tecnologías en uso.

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

## Endpoints (29 totales)

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

### Privados (16, requieren JWT)

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
| **POST** | **`/api/private/upload-image`** | **Subir imagen a GCS (multipart `file`; devuelve `{ url }`)** |
| GET | `/api/private/ops/metrics` | Métricas operativas |
| GET | `/api/private/ops/alerts` | Alertas operativas |
| GET | `/api/private/ops/health` | Estado de salud |
| GET | `/api/private/ops/history` | Historial de estados |
| GET | `/api/private/ops/summary` | Resumen para semáforo |

### Documentación

| Ruta | Descripción |
|------|-------------|
| `/swagger/*` | Swagger UI |

## Imágenes (upload y firma)

### Flujo de subida

1. El cliente (admin panel) hace `POST /api/private/upload-image` con `multipart/form-data` y campo `file`.
2. El backend valida tipo MIME (`image/jpeg`, `image/png`, `image/gif`, `image/webp`) y tamaño (≤ 5 MB).
3. El archivo se sube a GCS bajo el prefix `portfolio-images/{uuid}.{ext}` con `Cache-Control: private`.
4. La respuesta devuelve `{ "url": "https://storage.googleapis.com/<bucket>/portfolio-images/..." }`.
5. El cliente incluye esa URL en el campo `imageUrls` del payload de create/update, o la inserta en el body del editor.

### Validación de `imageUrls`

El campo `imageUrls` en experiencias y skills se valida en `sanitizer.ValidateURLSlice`:

- Solo se aceptan URLs con esquema **`http`** o **`https`**. Las `data:` URLs y cualquier otro esquema se descartan silenciosamente.
- Longitud máxima por URL: **2 048 caracteres** (`MaxImageURLLength`).
- Máximo **10 URLs** por entrada (`MaxImageURLCount`).

Los valores almacenados en base de datos son las URLs canónicas de GCS (`https://storage.googleapis.com/...`). Las **signed URLs** se generan en tiempo de lectura (endpoints de experiencias y skills) y tienen una expiración configurable (por defecto 7 días, `SIGNED_URL_EXPIRY_HOURS`).

### Seguridad del bucket

El bucket GCS es **privado** (acceso denegado vía Firebase client SDK, ver `storage.rules`). El acceso se realiza únicamente a través del backend con IAM/ADC. Las URLs firmadas V4 son de solo lectura y expiran automáticamente.

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
