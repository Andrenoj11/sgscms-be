# Deploy ke Vercel dengan Supabase

Arsitektur ini menggunakan Vercel Go Function untuk API, Supabase Postgres untuk data, dan Supabase Storage untuk upload gambar. Deployment VPS tetap dapat memakai konfigurasi `STORAGE_DRIVER=local`.

## 1. Buat project dan bucket Supabase

1. Buat project di Supabase dan simpan database password-nya.
2. Buka **Storage**, buat bucket bernama `cms-images`, lalu jadikan bucket **Public** agar URL gambar yang disimpan API dapat diakses publik.
3. Buka **Project Settings > API Keys** dan salin `service_role` key. Key ini hanya boleh disimpan sebagai environment variable backend, jangan pernah dikirim ke frontend.
4. Catat **Project URL**, misalnya `https://PROJECT_REF.supabase.co`.

Public URL bucket yang digunakan sebagai `UPLOAD_BASE_URL` adalah:

```text
https://PROJECT_REF.supabase.co/storage/v1/object/public/cms-images
```

## 2. Jalankan migration dan seed

Gunakan **Direct connection** dari tombol **Connect** di dashboard Supabase untuk migration dari komputer lokal. Pecah connection string tersebut ke variabel berikut di `.env` lokal:

```dotenv
DB_HOST=db.PROJECT_REF.supabase.co
DB_PORT=5432
DB_NAME=postgres
DB_USER=postgres
DB_PASSWORD=DATABASE_PASSWORD
DB_SSL_MODE=require
```

Kemudian jalankan:

```bash
go run ./cmd/migrate -direction up
go run ./cmd/seed
```

Migration sebaiknya tidak dijalankan otomatis oleh setiap Vercel Function karena beberapa instance dapat start bersamaan.

## 3. Siapkan koneksi runtime Vercel

Dari **Connect > Transaction pooler**, ambil nilai host, user, dan port. Transaction pooler biasanya memakai port `6543` dan user berbentuk `postgres.PROJECT_REF`.

Tambahkan environment variables berikut di **Vercel > Project Settings > Environment Variables**:

```dotenv
APP_NAME=SGS CMS API
APP_ENV=production
APP_DEBUG=false

DB_HOST=aws-0-REGION.pooler.supabase.com
DB_PORT=6543
DB_NAME=postgres
DB_USER=postgres.PROJECT_REF
DB_PASSWORD=DATABASE_PASSWORD
DB_SSL_MODE=require
DB_TIMEZONE=Asia/Jakarta
DB_QUERY_EXEC_MODE=simple_protocol
DB_MAX_OPEN_CONNS=2
DB_MAX_IDLE_CONNS=0

JWT_SECRET=SECRET_MINIMAL_32_KARAKTER
JWT_ISSUER=sgscms-api
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h

SIGNATURE_ENCRYPTION_KEY=BASE64_DARI_32_BYTE
SIGNATURE_MAX_AGE=5m
CORS_ALLOWED_ORIGINS=https://DOMAIN_FRONTEND_ANDA
LOGIN_RATE_LIMIT=5
LOGIN_RATE_WINDOW=1m

STORAGE_DRIVER=supabase
UPLOAD_BASE_URL=https://PROJECT_REF.supabase.co/storage/v1/object/public/cms-images
UPLOAD_MAX_IMAGE_SIZE=2097152
SUPABASE_URL=https://PROJECT_REF.supabase.co
SUPABASE_SERVICE_ROLE_KEY=SERVICE_ROLE_KEY
SUPABASE_STORAGE_BUCKET=cms-images
```

Jangan menambahkan `PORT` secara manual di Vercel. Vercel memberikan port dinamis ketika Function dijalankan dan aplikasi otomatis memprioritaskan nilai tersebut di atas `APP_PORT`.

Buat secret yang kuat, misalnya:

```bash
openssl rand -base64 48
openssl rand -base64 32
```

Gunakan hasil pertama untuk `JWT_SECRET` dan hasil kedua untuk `SIGNATURE_ENCRYPTION_KEY`.

## 4. Deploy ke Vercel

Push repository ke Git provider, lalu import repository tersebut di Vercel. Framework Preset dapat dibiarkan **Other**. File `api/index.go` menjadi entrypoint Go Function dan `vercel.json` meneruskan seluruh route ke Gin router.

Setelah deployment selesai, cek:

```bash
curl https://DOMAIN_VERCEL/health
```

Swagger UI ikut ter-deploy di Go Function yang sama dan tersedia di:

```text
https://DOMAIN_VERCEL/swagger/index.html
```

Tidak diperlukan project atau deployment Vercel terpisah untuk Swagger. Pada entrypoint serverless, nilai `host` dan `schemes` Swagger dikosongkan agar tombol **Try it out** otomatis mengirim request ke domain dan protokol deployment yang sedang dibuka. Raw OpenAPI JSON tersedia di:

```text
https://DOMAIN_VERCEL/swagger/doc.json
```

Setelah mengubah anotasi Swagger pada source code, generate ulang file dokumentasi sebelum push:

```bash
make swagger
```

Pastikan CLI `swag` yang dipakai kompatibel dengan versi dependency project.

## Catatan operasional

- Pilih region Vercel yang paling dekat dengan region project Supabase untuk mengurangi latency.
- Gunakan Transaction pooler untuk request aplikasi Vercel, tetapi Direct connection untuk migration.
- `DB_QUERY_EXEC_MODE=simple_protocol` diperlukan karena transaction pooler tidak mendukung prepared statement.
- Rate limiter aplikasi saat ini tersimpan di memory setiap Function instance. Untuk rate limit global lintas instance, pindahkan state ke layanan eksternal seperti Redis.
- Deployment preview membutuhkan `CORS_ALLOWED_ORIGINS` sendiri bila frontend preview perlu mengakses API.
