# Evaluasi Project TripNest

Review kode per 19 Agustus 2026 — commit `5d5ce8e`.
Cakupan: 5 service Go aktif (~4.700 baris), frontend Next.js 16, Docker Compose, migrasi SQL.

---

## Ringkasan

Yang sudah bagus dan tidak sepele:

- Pemisahan service, layering `handler → service → repository → model` konsisten di semua service.
- Outbox table + `processed_messages` untuk idempotensi consumer — konsepnya sudah benar.
- `UpdateStatusIfCurrent` (compare-and-set di level SQL) untuk transisi status booking — ini cara yang tepat, bukan read-modify-write.
- Verifikasi signature SHA-512 webhook Midtrans + idempotency key `order_id:transaction_status`.
- Worker expiry booking dengan `RETURNING id` sehingga tidak ada race antar instance.
- `ApplyWebhookResult` memakai transaksi Postgres.
- Graceful shutdown di semua service.
- README-nya jujur — bagian "Limitations & Edge Cases" menyebut beberapa masalah nyata.

Masalah utamanya bukan struktur, tapi **lapisan otorisasi yang praktis tidak ada** dan **beberapa klaim arsitektur di README yang belum benar-benar diimplementasikan di kode**. Untuk portofolio ini sudah kuat; untuk dipakai orang, ada 6 hal yang harus dibereskan dulu.

---

## 1. Kritis — harus dibereskan sebelum apa pun

### 1.1 Harga booking dikirim dari browser

`frontend/app/page.tsx:39-45` mengirim harga lewat query string, `app/booking/page.tsx:75` meneruskannya sebagai `total_amount`, dan `booking-service/internal/handler/handler.go:41` hanya memvalidasi `TotalAmount > 0`.

Artinya siapa pun bisa mengubah URL jadi `/booking?tripId=...&price=1000` dan membayar trip 5 juta seharga Rp1.000. Midtrans akan memproses pembayaran itu dan booking akan `CONFIRMED`.

**Perbaikan:** booking-service harus mengambil harga dari sumber kebenaran (search-service / trip). Request dari client cukup `resource_id` saja, harga dihitung server-side.

### 1.2 `user_id` dipercaya dari client

Payload `POST /bookings` berisi `user_id` yang diambil dari `localStorage`. Tidak ada middleware JWT di booking-service — `cfg.JWTSecret` dimuat di `config.go:29` tapi **tidak pernah dipakai di mana pun**.

Konsekuensi: siapa saja bisa membuat booking atas nama user lain, dan `GET /bookings/{id}` bisa dibaca siapa saja tanpa token.

**Perbaikan:** middleware JWT di setiap service, `user_id` diambil dari claim token, bukan dari body.

### 1.3 Admin panel tanpa autentikasi sama sekali

Tiga lapis masalah bertumpuk:

1. `frontend/app/admin/login/page.tsx:24` — kredensial `admin` / `admin123` hardcoded di JavaScript client. Terbaca siapa pun yang membuka DevTools.
2. `localStorage.setItem('admin_logged_in', 'true')` — **tidak pernah dicek sebagai guard**. Saya grep seluruh frontend: key ini hanya di-*set* dan di-*remove*, tidak ada satu pun `if` yang memeriksanya. `/admin/*` bisa dibuka langsung tanpa login.
3. Endpoint backend-nya sendiri publik: `POST/PUT/DELETE /trips` dan `POST /seed` (search-service), `POST /inventory` + `PATCH /inventory/{id}` (inventory-service), `GET /payments` (payment-service) — semuanya tanpa autentikasi.

`GET /payments` mengembalikan seluruh baris payment termasuk `snap_token` dan `raw_notification`. Itu kebocoran data pembayaran.

**Perbaikan:** hapus login admin palsu, pakai `role` yang sudah ada di tabel `users` + JWT claim, dan pasang middleware `RequireRole("admin")` di semua endpoint tulis.

### 1.4 `GET /payments/snap-token` bisa memicu transaksi Midtrans

Endpoint ini publik dan — kalau payment belum ada — akan **membuat transaksi Snap baru** (`service.go:130-150`). Seseorang bisa mengenumerasi booking ID dan menghabiskan kuota Midtrans, sekaligus mengambil snap token milik orang lain.

### 1.5 `backend/user-service/.env` ter-commit ke git

`git ls-files` menunjukkan file ini terlacak, berisi `JWT_SECRET` dan `DATABASE_URL`. Tidak ada `.gitignore` di root repo sama sekali — `.env` di root dan `backend/.env` (yang berisi `MIDTRANS_SERVER_KEY`) belum ter-commit murni karena keberuntungan.

**Perbaikan:** buat `.gitignore` root, `git rm --cached backend/user-service/.env`, **rotasi Midtrans server key** kalau pernah ter-push, dan sediakan `.env.example`.

### 1.6 JWT secret default lemah

`env.GetString("JWT_SECRET", "super-secret-key")` di user-service dan booking-service, dan `docker-compose.yml` mengesetnya ke `dev-secret`. Kalau ini pernah naik ke server, siapa pun bisa menandatangani token admin sendiri. Service harus `log.Fatal` kalau `JWT_SECRET` kosong di mode produksi — persis seperti yang sudah dilakukan payment-service untuk Midtrans key.

---

## 2. Arsitektur — klaim README vs kode

### 2.1 Outbox pattern belum transaksional (ini yang paling penting)

README menyatakan: *"the database commit and the intent to publish stay consistent"*. Kodenya tidak begitu.

`booking-service/internal/service/service.go:56-63`:

```go
if err := s.repo.Create(ctx, booking); err != nil { return nil, err }
// ...
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := s.repo.AddOutboxEvent(ctx, "booking.created", booking.ID, event); err != nil {
        fmt.Printf("Failed to enqueue booking.created outbox event: %v\n", err)
    }
}()
```

Insert booking dan insert outbox adalah **dua transaksi terpisah**, dan yang kedua dijalankan di goroutine yang error-nya cuma di-`Printf`. Kalau proses mati di antara keduanya, atau insert outbox gagal, booking-nya ada tapi event `booking.created` hilang selamanya — user punya booking `PENDING_PAYMENT` yang tidak akan pernah dapat snap token dari jalur event, dan akan `EXPIRED` diam-diam.

Saya cek `booking-service/internal/repository/repository.go`: tidak ada satu pun pemanggilan `BeginTx` di seluruh file. Hal yang sama terjadi di payment-service — `ApplyWebhookResult` punya transaksi sendiri, lalu `AddOutboxEvent` dipanggil terpisah setelahnya (`service.go:236`).

**Perbaikan:** ubah signature repository agar menerima `pgx.Tx`, lalu insert entitas + insert outbox dalam satu `BeginTx`/`Commit`. Ini perubahan ~50 baris dan langsung membuat klaim README jadi benar.

### 2.2 Idempotensi ditandai *sebelum* diproses — event bisa hilang

Pola ini berulang di booking, payment, dan inventory service:

```go
processed, err := repo.MarkMessageProcessed(ctx, group, msg.Topic, string(msg.Key))
if !processed { continue }
// ... baru di sini svc.ConfirmBooking(ctx, event.BookingID)
```

Kalau `ConfirmBooking` gagal (DB down sesaat), errornya hanya di-log, tapi baris `processed_messages` sudah ter-commit. Retry berikutnya akan di-skip sebagai "duplikat". Ini **at-most-once**, bukan at-least-once — pembayaran sukses tapi booking tidak pernah `CONFIRMED`.

Diperparah oleh `payment-service/cmd/api/main.go:92` yang memanggil `go svc.ProcessPayment(ctx, event)` — fire-and-forget, return error dibuang total.

**Perbaikan:** proses dulu, tandai `processed_messages` di transaksi yang sama dengan efek bisnisnya, dan baru commit offset Kafka.

### 2.3 Consumer mati permanen saat error

`payment-service/cmd/api/main.go:78` dan keempat consumer di inventory-service memakai `break` saat `ReadMessage` error, bukan `continue`. Satu blip jaringan ke Kafka = consumer mati sampai container di-restart. Booking-service sudah benar (`continue`) — samakan saja polanya.

### 2.4 Tidak ada pengecekan inventory sebelum booking

Alurnya: booking dibuat → `booking.created` → inventory-service coba `ReserveSlot` → kalau slot habis, `service.go:47` hanya menulis `log.Printf("Reservation no-op")`.

Booking tetap jalan ke pembayaran. User bisa membayar penuh untuk trip yang sudah sold out, dan tidak ada mekanisme refund atau pembatalan otomatis. Untuk sistem booking, ini bug domain yang paling serius setelah masalah harga.

**Perbaikan:** ada dua opsi — booking-service memanggil inventory secara sinkron sebelum insert (lebih sederhana, kopling lebih ketat), atau inventory-service menerbitkan `inventory.rejected` yang di-consume booking-service untuk membatalkan booking (saga compensation yang benar). Opsi kedua lebih sesuai dengan arsitektur yang sudah ada — dan `AddOutboxEvent(..., "inventory.reserved", ...)` sudah ada, tinggal ditambah jalur reject-nya.

### 2.5 Dua sumber kebenaran untuk stok

`search-service` (MongoDB) punya `available_slots`, `inventory-service` (Postgres) juga punya `available_slots`. Tidak ada yang menyinkronkan keduanya. Frontend menampilkan angka dari search-service — yang **tidak pernah berkurang** saat ada booking. Angka "Sisa N kursi" di `TripCard.tsx` selalu salah.

### 2.6 Cache Redis tidak pernah di-invalidasi

`cached_repository.go` — `Create`, `Update`, `Delete` diteruskan langsung ke Mongo tanpa menghapus key `search:*`. Setelah admin mengubah trip, hasil search masih basi sampai TTL 60 detik habis.

### 2.7 Outbox worker tidak aman untuk multi-instance

`FetchPendingOutboxEvents` memakai `SELECT ... WHERE status='PENDING' ORDER BY created_at LIMIT $1` tanpa `FOR UPDATE SKIP LOCKED`. Jalankan 2 replika booking-service dan event yang sama akan dipublikasikan dua kali. Idempotensi consumer menyelamatkan sebagian, tapi ini menghapus salah satu klaim "horizontally scalable" di README.

Selain itu `MarkOutboxEventFailed` hanya menaikkan `retry_count` tanpa pernah menset status `FAILED` — event yang rusak permanen (misal payload tidak bisa di-unmarshal) akan di-retry setiap 2 detik selamanya. Perlu batas maksimum retry + dead-letter, dan exponential backoff.

### 2.8 Error handling berbasis pencocokan string

`payment-service/internal/service/service.go:300` dan `handler.go:140`:

```go
return strings.Contains(msg, "conflict") ||
       strings.Contains(msg, "duplicate") ||
       strings.Contains(msg, "not found") || ...
```

Rapuh dan berbahaya. `isIgnorableWebhookError` menelan `"not found"` — jadi kalau webhook masuk untuk `midtrans_order_id` yang tidak ada di DB, service mengembalikan `200 OK` ke Midtrans dan pembayaran itu hilang tanpa jejak. Ganti dengan sentinel error (`errors.Is`) dan pemeriksaan `pgconn.PgError.Code == "23505"`.

### 2.9 `api-gateway` kosong

`backend/api-gateway/` hanya berisi `go.mod` dan `go.sum` kosong, tapi tetap terdaftar di `go.work` (begitu juga `notification-service` dan `recommendation-service`). Ini membuat `go build ./...` dari root gagal, dan browser jadi harus bicara langsung ke 4 port berbeda dengan CORS di-hardcode ke `http://localhost:3000` di setiap service.

---

## 3. Frontend

| Masalah | Lokasi | Dampak |
|---|---|---|
| Riwayat booking disimpan di `localStorage` (`my_bookings`) | `app/my-bookings/page.tsx:20` | Hilang saat clear browser, tidak sinkron antar device, tidak per-user. Backend sudah punya `GetByUserID` tapi tidak ada endpoint yang mengeksposnya. |
| Token JWT di `localStorage` | `lib/api.ts:26` | Terbaca oleh XSS. Idealnya httpOnly cookie. |
| `http://localhost:8082` hardcoded | `app/api/payments/snap-token/route.ts:11` | Route handler ini akan rusak di production. Satu-satunya tempat yang tidak pakai env var. |
| Tidak ada `middleware.ts` | — | Tidak ada route protection sama sekali; semua guard dilakukan client-side (atau tidak sama sekali). |
| Tidak ada refresh token / auto-logout | — | Token 24 jam expired diam-diam, user melihat error generik. |
| Tidak ada state management terpusat | — | Auth state dibaca ulang dari `localStorage` di tiap komponen; `login/page.tsx:31` memakai `window.location.href` untuk memaksa reload navbar. |
| `debug` dan `es-shim-unscopables` di `dependencies` | `package.json` | Ini transitive deps yang tidak seharusnya eksplisit. |
| Folder `node_modules/@babel 2`, `@babel 3`, `react-hot-toast 2/3`, `package-lock 2.json` | — | Artefak duplikasi iCloud Drive. Repo di iCloud + `node_modules` adalah kombinasi yang bermasalah. Pindahkan project keluar dari iCloud atau exclude foldernya. |

Halaman `/search` dan `/bookings` juga ada tapi filter destination/date/price dari `SearchParams` backend belum dipakai penuh — frontend memfilter client-side di `page.tsx:31` (`trips.filter(...)`), padahal endpoint `/search` sudah mendukung pagination dan filter harga di server.

---

## 4. Kualitas & Operasional

**Tidak ada satu pun test.** Nol file `*_test.go`, nol `*.test.ts`. Untuk sistem terdistribusi dengan saga dan state machine pembayaran, ini yang paling mahal dampaknya jangka panjang — tidak ada cara aman untuk merefaktor 9 poin di atas.

**Tidak ada CI.** Tidak ada `.github/`. Minimal: `go vet`, `go test ./...`, `golangci-lint`, `npm run build`.

**Observability nol.** Semua pakai `log.Printf` tanpa struktur, tanpa correlation ID, tanpa level. Tidak ada `/metrics`, tidak ada tracing. Menelusuri satu booking melewati 4 service saat ini berarti `grep` manual di 4 container.

**Log membocorkan data.** `payment-service/internal/service/service.go:87` mencetak `MIDTRANS PAYLOAD: %+v` (berisi data customer), `midtrans/client.go:88` mencetak seluruh response Midtrans mentah, dan `fetchBooking` mencetak body booking lengkap.

**Error internal dikirim ke client.** `utils.WriteError(w, 500, err)` mengirim `err.Error()` apa adanya — termasuk pesan error Postgres. Bocorkan struktur skema.

**Register mengembalikan 500 untuk email duplikat** (`user-service/handler.go:36`) — seharusnya 409. Dan tidak ada validasi format email atau kekuatan password sama sekali.

**Tidak ada rate limiting** di endpoint `/login` — brute force bebas.

**Kafka pakai Zookeeper** (`cp-kafka:7.8.0` + `cp-zookeeper`). Sejak Kafka 3.x, KRaft mode lebih sederhana dan menghilangkan satu container.

**Tidak ada healthcheck** di service container Compose, hanya di kafka/postgres. `depends_on` tidak menunggu service siap.

**Docker image**: `alpine:latest` tanpa non-root user, tanpa pin versi, dan `RUN go mod tidy` di dalam Dockerfile (build tidak reproducible — seharusnya `go mod download` saja dengan `go.sum` yang sudah benar).

**Migrasi**: `000001_create_bookings_table.up.sql` sudah membuat `expires_at`, lalu `000002` menambahkannya lagi. Idempoten jadi tidak error, tapi menandakan migrasi lama pernah diedit setelah diterapkan. Juga ada dua mekanisme migrasi paralel — `golang-migrate` di dalam service dan `scripts/db-setup.sh` yang punya tabel `schema_migrations` sendiri. Pilih satu.

**Uang disimpan sebagai `float64`.** `total_amount DECIMAL(10,2)` di DB tapi `float64` di Go, dan `int64(math.Round(event.TotalAmount))` saat dikirim ke Midtrans. Untuk IDR kebetulan aman, tapi tidak ada field `currency` di mana pun — pakai integer minor units (sen) atau `decimal.Decimal`.

**Data customer Midtrans hardcoded**: `"TripNest Customer"`, `user@tripnest.com`, `08123456789` (`service.go:78-83`). Struk dan email dari Midtrans akan salah semua. `booking.created` sudah membawa `user_id` — tinggal ambil profil user.

**Snap transaction tidak punya `expiry`** yang selaras dengan `expires_at` booking, dan tidak ada `item_details`.

---

## 5. Roadmap Pengembangan

### Tahap 1 — Amankan (1–2 minggu)

Ini yang harus selesai sebelum fitur baru apa pun.

1. `.gitignore` root + `git rm --cached backend/user-service/.env` + rotasi Midtrans server key.
2. Package `common/auth` berisi middleware JWT + `RequireRole`. Pasang di semua service.
3. Harga booking diambil server-side dari trip, bukan dari request body. `user_id` dari JWT claim.
4. Hapus admin login hardcoded — pakai `role='admin'` di tabel `users`.
5. Proteksi `GET /payments`, `POST /seed`, semua endpoint tulis trip & inventory.
6. `JWT_SECRET` wajib (fatal kalau kosong), hilangkan default.
7. Rate limit `/login` dan `/register`.

### Tahap 2 — Perbaiki fondasi (2–3 minggu)

8. Outbox benar-benar transaksional (satu `BeginTx` untuk entitas + outbox).
9. Idempotensi: proses dulu, tandai belakangan, dalam satu transaksi.
10. `FOR UPDATE SKIP LOCKED` di outbox worker + max retry + dead-letter table + backoff.
11. Ganti `break` jadi `continue` di consumer payment & inventory; hapus `go svc.ProcessPayment(...)`.
12. Ganti pencocokan error berbasis string dengan sentinel error + `pgconn.PgError`.
13. Cek inventory sebelum booking (via `inventory.rejected` + kompensasi).
14. Invalidasi cache Redis saat trip berubah.
15. Sinkronkan `available_slots` — satu sumber kebenaran (inventory-service), search-service consume event untuk update read model. Ini yang membuat klaim CQRS di README jadi nyata.
16. Endpoint `GET /bookings?user_id=me` (dari JWT) menggantikan `localStorage`.

### Tahap 3 — Kualitas (2–3 minggu)

17. Unit test service layer dengan mock repository — mulai dari `mapMidtransStatus`, `isSignatureValid`, state machine booking. Ini yang paling bernilai per baris test.
18. Integration test dengan `testcontainers-go` (Postgres + Kafka nyata) untuk alur end-to-end booking → payment → confirm.
19. GitHub Actions: `go vet`, `golangci-lint`, `go test -race ./...`, `npm run build`.
20. Structured logging (`log/slog`) + correlation ID yang mengalir lewat header HTTP dan Kafka message header.
21. Prometheus `/metrics` + Grafana. Metrik yang berharga: outbox lag, consumer lag, durasi booking→confirmed, tingkat kegagalan webhook.
22. OpenTelemetry tracing lintas service.
23. Healthcheck `/health/ready` yang benar-benar mengecek DB & Kafka, bukan sekadar `200 OK`.

### Tahap 4 — Fitur yang membuat project ini menonjol

24. **API Gateway** — isi folder yang sekarang kosong. Satu entry point, JWT divalidasi sekali di gateway, rate limiting terpusat, CORS satu tempat. Ini sekaligus menyelesaikan masalah frontend harus tahu 4 port.
25. **notification-service** — sudah ada folder kosongnya dan sudah ada topic `booking.created` / `payment.success` / `booking.expired`. Email konfirmasi booking + e-ticket PDF. Ini menunjukkan nilai arsitektur event-driven: service baru cukup subscribe, tanpa mengubah service lain sama sekali. Demonstrasi terbaik dari desain yang sudah kamu buat.
26. **Refund flow** — Midtrans punya refund API. Cancel booking yang sudah `CONFIRMED` → refund → `payment.refunded` → inventory release.
27. **recommendation-service** — folder kosong ketiga. Bisa sesederhana "trip populer berdasarkan jumlah booking", consume `payment.success` untuk membangun ranking.
28. **Booking multi-slot** — sekarang satu booking = satu slot. Tambah `quantity` + `passenger_details`.
29. **Search yang sebenarnya** — MongoDB text index atau Elasticsearch, filter tanggal, sorting, geospatial.
30. **Deployment** — Kubernetes manifest / Helm chart, atau minimal Compose production dengan secret management. Ini yang paling sering ditanya reviewer untuk project microservices.

---

## Catatan Penutup

Kalau tujuannya portofolio: yang paling meningkatkan nilai per jam kerja adalah **Tahap 1 + poin 8, 9, 17, 19** — keamanan dasar, outbox yang benar-benar transaksional, dan bukti bahwa kamu menulis test. Setelah itu **notification-service** (poin 25), karena itu satu-satunya hal yang benar-benar membuktikan arsitektur event-driven-mu berfungsi.

Kalau tujuannya produksi: Tahap 1 sampai 3 tidak bisa dilewati.

Satu hal yang layak diperbaiki segera dan cuma butuh 5 menit: **pindahkan repo ini keluar dari iCloud Drive**. Folder `node_modules/@babel 2`, `@babel 3`, `react-hot-toast 2`, `react-hot-toast 3`, dan `package-lock 2.json` adalah konflik sinkronisasi iCloud, dan itu bisa menyebabkan build yang gagal secara acak dan sulit didiagnosis.
