# KampusOps

SIAKAD multi-kampus berbasis SaaS: dari PMB sampai lapor PDDikti. Backend memakai **Go Fiber**, frontend memakai **Vue 3 + Pinia**.

Pembedanya bukan jumlah modul, tetapi cara sistem menegakkan aturan bisnis kampus yang rumit:
- KRS rebutan kuota tanpa overbooking
- Prasyarat lewat ekuivalensi
- Cuti akademik dan batas masa studi
- Koreksi nilai setelah dikunci
- Cicilan UKT dan webhook pembayaran ganda
- Validasi data sebelum lapor PDDikti

![Dashboard operator](screenshots/50-operator-dashboard.png)

| Dokumen | Isi |
|---|---|
| [BRD](BRD%20—%20Sistem%20Operasional%20Kampus%20(KampusOps).md) | Kebutuhan bisnis, edge case EC-01..17, aturan PV-01..08 |
| [docs/FEATURES.md](docs/FEATURES.md) | **Tur fitur per peran dengan screenshot** |
| [docs/DESIGN_SYSTEM.md](docs/DESIGN_SYSTEM.md) | Token, komponen, dan aturan konsistensi UI |
| [docs/API_CONTRACT.md](docs/API_CONTRACT.md) | Kontrak API v1 (sumber kebenaran antara backend dan frontend) |
| [backend/README.md](backend/README.md) | Arsitektur backend, keputusan teknis, test, dan load test |

## Menjalankan

### Satu perintah (Docker)

```bash
docker compose up --build
```

Perintah ini menjalankan Postgres 16, Redis 7, API (`:8080`, migrasi otomatis), worker, dan frontend (`:5173`). Isi data demo:

```bash
docker compose run --rm seed
```

Lalu buka http://localhost:5173.

### Lokal tanpa Docker

```bash
# Postgres + Redis lokal sudah berjalan
cd backend
export DATABASE_URL="postgres://<user>@localhost:5432/kampusops?sslmode=disable"
go run ./cmd/seed          # migrasi + data sintetis
go run ./cmd/api           # :8080

cd ../frontend
npm install
npm run dev                # :5173, /api di-proxy ke :8080
```

### Akun demo

Kampus `UNIKA`, kata sandi `password123`. Di halaman login, tombol **Masuk cepat** langsung masuk sebagai peran yang dipilih.

| Peran | Login |
|---|---|
| Mahasiswa (aktif, lunas) | `2024110001` |
| Mahasiswa (belum bayar, KRS diblokir) | `2024110002` |
| Dosen + dosen wali + kaprodi | `0011223344` |
| Dosen | `0055667788` |
| Operator akademik | `baak@unika.ac.id` |
| Keuangan | `keuangan@unika.ac.id` |
| Admin kampus | `admin@unika.ac.id` |
| Super admin (vendor) | `super@kampusops.id` |
| Calon mahasiswa | `calon@mail.com` |

Kampus kedua, `POLTEK` (WITA), memakai kode prodi dan NIM yang sama untuk membuktikan isolasi antar tenant (EC-16).

## Arsitektur

```
frontend/  Vue 3 + TS + Vite + Pinia + Tailwind (token terkunci) + Headless UI
  src/design-system/   token, status map, formatter, komponen K*, chart SVG
  src/views/           satu folder per domain (mahasiswa, dosen, akademik, keuangan, pmb, pelaporan, admin)
  e2e/                 Playwright: smoke flow + generator screenshot
backend/   Go Fiber modular monolith (lihat backend/README.md)
  cmd/api, cmd/worker, cmd/seed
  internal/<domain>/   handler → service (semua aturan bisnis) → repository (pgx, selalu tenant_id)
  migrations/
docs/      kontrak API, design system, panduan fitur
screenshots/
```

**Keputusan utama:**
- **Kontrak API dulu.** `docs/API_CONTRACT.md` ditulis sebelum kode, sehingga backend dan frontend bisa dibangun paralel. Kode error bisnis (`KRS_QUOTA_FULL`, dst.) dipetakan ke satu kalimat di frontend.
- **Aturan di service layer.** Handler dan worker hanya memanggilnya, jadi setiap aturan diuji sekali dan berlaku di semua jalur.
- **UI konsisten secara struktural, bukan lewat disiplin.**
  - Palet Tailwind diganti total dengan token.
  - Status hanya dirender lewat `<KStatus>` dari satu peta.
  - Semua layar master data dirender oleh satu `ResourcePage` berbasis konfigurasi.

## Test

```bash
cd backend && go test -short ./...          # unit test aturan bisnis
cd backend && go test ./...                 # + integrasi testcontainers (EC-01, EC-04, EC-16)
cd frontend && npm test                     # Vitest: format, status map, hitung nilai (EC-10), komponen
cd frontend && npm run e2e                  # Playwright smoke flow (butuh stack + seed)
k6 run backend/loadtest/krs_submit.js       # 500 submit KRS bersamaan
```

## Screenshots

Semua screenshot ada di [`screenshots/`](screenshots) dan bisa dibuat ulang dari data seed:

```bash
cd frontend
BASE_URL=http://localhost:5173 npm run screenshots
```

Script login ke setiap peran lewat API, membuka setiap fitur, lalu mengambil gambar desktop (1440 px @2x), tema gelap, mobile (390 px @3x), dan command palette.
