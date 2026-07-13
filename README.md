# SGS Law Firm CMS backend

Golang modular-monolith backend for News, Our Team, media, authentication, RBAC, audit trails, SEO migration fields, and trusted signed requests. The domain and application packages do not depend on HTTP, PostgreSQL, or object-storage implementations.

## Run locally

Requirements: Docker Compose, or Go 1.22 plus PostgreSQL 16.

```sh
docker compose up --build
```

The API listens on `http://localhost:8080`. On the first start it applies the embedded database migration and creates the development administrator:

- Email: `admin@example.com`
- Password: `change-me-now`

Change or remove the `BOOTSTRAP_ADMIN_*` values before using a shared environment. Bootstrap is idempotent and does not overwrite an existing user's password.

Without Docker, copy `.env.example` to `.env`, export its values, start PostgreSQL, and run:

```sh
go run ./cmd/api
```

Useful checks:

```sh
go test ./...
go vet ./...
```

## API

Public read-only routes require no authentication and only expose currently published/active content:

```text
GET /api/v1/public/news
GET /api/v1/public/news/featured
GET /api/v1/public/news/categories
GET /api/v1/public/news/{slug}
GET /api/v1/public/team
GET /api/v1/public/team/{slug}
GET /api/v1/public/practice-areas
GET /api/v1/public/practice-areas/{slug}/team
```

List endpoints accept `page`, `limit`, `search`, `category`, `tag`, `status`, and `featured` where relevant. Public responses have short HTTP cache headers.

Administrative routes live under `/api/v1/admin`. Login creates an HttpOnly session cookie and returns a CSRF token. Send that token as `X-CSRF-Token` on every administrative write request. Each route also checks its specific permission in the application boundary.

```sh
curl -i -c cookies.txt -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"change-me-now"}' \
  http://localhost:8080/api/v1/admin/auth/login
```

The admin resources are `/news`, `/team`, `/categories`, `/tags`, `/media`, `/users`, `/roles`, `/audit-logs`, `/redirects`, and `/api-clients`. Updates and delete/restore operations require the current `version`; stale versions return HTTP 409. Upload media with multipart fields `file`, `altText`, and `caption`. API-client secrets are returned exactly once when created.

Health checks:

```text
GET /health/live
GET /health/ready
```

## Trusted request signing

`POST /api/v1/internal/news/publish-due` is the signed server-to-server endpoint. It requires `news.publish` on the API client and these headers:

```text
X-Key-ID
X-Timestamp             RFC3339
X-Nonce
X-Signature-Version     v1
X-Body-SHA256           lowercase hex
X-Signature             lowercase HMAC-SHA256 hex
```

The canonical input, joined by newlines, is:

```text
version
key ID
timestamp
nonce
uppercase HTTP method
request URI including query
content type
SHA-256 body hash
```

Nonces are atomically registered in PostgreSQL after successful verification. A valid signature still needs the endpoint permission. API-client secrets must only be provisioned to trusted servers; never return them to browser code.

## Storage and production notes

Local development stores validated uploads under `var/media`. Set `S3_ENDPOINT` and the related `S3_*` variables to use MinIO, Cloudflare R2, AWS S3-compatible endpoints, or similar object storage. Uploaded JPEG, PNG, and WebP files are checked by detected content and decoded dimensions, receive random keys, and never use the original filename as their storage path.

For production:

- use TLS at the reverse proxy and set `SESSION_SECURE=true`;
- set restricted CORS at the edge for the separate admin origin;
- store database and S3 credentials in a secret manager;
- set a strong, stable `API_CLIENT_ENCRYPTION_KEY` through a secret manager (changing it invalidates encrypted signing secrets);
- make the media bucket or CDN URL readable only as intended;
- run PostgreSQL backups and object versioning/replication;
- provision API-client secrets using application-level encryption or a secret manager;
- disable automatic migrations if deployments run migrations separately (`AUTO_MIGRATE=false`).

## Structure

```text
cmd/api                         composition, scheduler, graceful shutdown
internal/domain                 entities, rules, repository interfaces
internal/application            auth, content, media, signing use cases
internal/infrastructure         PostgreSQL and object-storage adapters
internal/delivery/httpapi       routes, middleware, response mapping
migrations                      versioned PostgreSQL schema
```

The initial schema intentionally extends the supplied ERD with BRD-required sessions, permissions, role permissions, audit logs, redirects, SEO/migration fields, soft-delete timestamps, scheduled publication, and optimistic-lock versions.
