# Bitácora

Registro de sesiones de trabajo sobre el portafolio (`front-end` + `go-api-rest`).
Cada entrada resume problemas encontrados, causas y decisiones. La misma entrada vive en ambos repos.

---

## 2026-10-09 — UX de contenido, imágenes privadas y Open Graph

Sesión en paralelo con Codex (Codex: SEO vs SPA / SSG; esta sesión: eventos de UX).

### 1. Vista previa rota al subir imágenes en el admin

- **Síntoma:** al subir una imagen, la URL remota se creaba pero la vista previa no cargaba.
- **Causa:** el bucket de GCS es privado (403 sin firma). `POST /api/private/upload-image` devolvía la URL canónica sin firmar; la API solo firmaba URLs al leer contenido.
- **Solución:** el upload devuelve `url` (canónica, se persiste) y `previewUrl` (firmada, temporal). El admin renderiza la firmada y guarda la canónica. El backend quita parámetros de firma de `imageUrls` antes de persistir.
- **Conclusión:** con bucket privado, distinguir siempre *URL para persistir* de *URL para mostrar*.

### 2. Experiencias, capacidades y artículos se veían iguales

- **Síntoma:** una capacidad aparecía como artículo y como capacidad en el landing; `/portfolio` la mostraba como experiencia; el admin la etiquetaba "Experiencia".
- **Causa:** había cinco reglas de clasificación distintas; el slider del blog buscaba palabras ("arquitectura") en el cuerpo del texto.
- **Solución:** clasificador único por tags (`utils/content-kind.ts`, precedencia blog > skill > portfolio > experiencia) y una representación por tipo:
  - Capacidad → tile compacto con stack (sin imagen protagonista).
  - Muestra → tarjeta con galería.
  - Experiencia → línea de tiempo con fecha.
- **Conclusión:** clasificar por tags, nunca por contenido libre; cada tipo comunica algo distinto y debe verse distinto.

### 3. Open Graph no funcionaba al compartir artículos

Tres causas encadenadas, resueltas en orden:

1. **Sin meta tags en el HTML estático:** solo existía `og:image`, y el build eliminaba las imágenes GCS para no congelar firmas. Los crawlers no ejecutan JS. → Meta tags completas (`og:*`, `twitter:*`, `description`, `canonical`) generadas en `getStaticProps`.
2. **Imagen inaccesible / inadecuada:** URLs firmadas expiran; `helmet` añade `Cross-Origin-Resource-Policy: same-origin`, que impide mostrarla fuera del origen; la imagen era vertical y pesaba 3.9 MB. → Endpoint público `GET /api/media/:name` (solo nombres `uuid.ext`), con `CORP: cross-origin`, `robots.txt` que permite `/api/media/` y variante `?variant=og` (recorte 1.91:1, 1200×630 JPEG, ~240 KB).
3. **Dominio canónico roto:** `og:url`/canonical/sitemap apuntaban a `yonathangutierrez.dev` (apex), que responde 404 porque no está conectado en Firebase Hosting. Facebook sigue `og:url`. → Canónico pasa a `https://www.yonathangutierrez.dev` (`SITE_URL` en `utils/seo.ts`, sitemap y robots).

- **Conclusión:** validar OG con el HTML que recibe el crawler (`curl -A facebookexternalhit/1.1`), no con DevTools; y tras corregir, forzar "Scrape Again" en el Sharing Debugger de Facebook.

### Documentación

- Swagger regenerado (`swag init`): incluye `/api/media/{name}`, `previewUrl` del upload y `/api/private/ops/redeploy`.

### Despliegue

- Frontend: `firebase deploy --only hosting` (la CLI usa las credenciales ADC de gcloud).
- API: `gcloud builds submit` + `gcloud run deploy porfolio-api` (revisión `porfolio-api-00014-jlw`).
- Los repos no versionan workflows de CI: el `ci.yml` vive en la carpeta del workspace, así que un push no despliega.

### Pendientes

- Conectar el dominio apex en Firebase Hosting como redirección a `www`.
- Gate de cobertura del backend en 77.7 % (< 80 %): faltan tests de `TriggerRedeploy`, `Logout`, `GetCurrentUser` y `GetOps*`.
- `pages/blog/*` mantiene su propio `BLOG_TAGS` (coincidencia por substring); migrar a `resolveContentKind`.
- `go get @latest` sube la directiva `go` a 1.26 y rompe el Dockerfile (`golang:1.25-alpine`); fijar versiones compatibles.
