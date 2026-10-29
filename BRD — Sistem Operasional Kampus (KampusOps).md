# BRD — Sistem Operasional Kampus (KampusOps)

Oct 28, 2026 · @Diamond

## 1. Ringkasan eksekutif

KampusOps adalah sistem informasi akademik (SIAKAD) multi-kampus berbasis SaaS yang mencakup siklus mahasiswa dari pendaftaran (PMB) sampai kelulusan dan pelaporan PDDikti. Proyek ini dibangun sebagai portofolio untuk posisi Software Engineer di SEVIMA, dengan Go Fiber di backend dan Vue 3 + Pinia di frontend.

Pembedanya bukan jumlah modul, tetapi cara sistem menangani aturan bisnis kampus yang rumit: KRS rebutan kuota, prasyarat mata kuliah, cuti akademik, perubahan nilai setelah dikunci, cicilan UKT, dan validasi data sebelum lapor PDDikti.

**Masalah yang diselesaikan**

- Data akademik kampus tersebar di Excel dan aplikasi terpisah, sehingga sering tidak sinkron antara bagian akademik, keuangan, dan operator PDDikti.
- Aturan akademik (batas SKS per IPS, prasyarat, status mahasiswa) sering dicek manual oleh operator dan rawan salah.
- Pelaporan PDDikti ditolak karena data tidak konsisten, dan baru ketahuan menjelang batas waktu pelaporan.
- Masa KRS menimbulkan lonjakan trafik dan rebutan kursi kelas yang membuat sistem lambat atau data kuota jebol.

## 2. Tujuan, ruang lingkup, dan metrik keberhasilan

Tujuan utama: menunjukkan pemahaman proses bisnis kampus dan kemampuan membangun produk SaaS yang benar secara aturan, aman saat trafik tinggi, dan mudah dipakai operator.

**Tujuan bisnis**

1. Satu sumber data akademik dan keuangan per kampus (single source of truth).
2. Aturan akademik ditegakkan sistem, bukan dicek manual.
3. Data siap lapor PDDikti dengan validasi sebelum kirim.
4. Satu instance melayani banyak kampus (multi-tenant), seperti model SaaS.

**Dalam ruang lingkup (MVP)**

- Multi-tenant dan RBAC
- Master data: fakultas, prodi, dosen, mahasiswa, ruang, periode akademik
- PMB sederhana sampai mahasiswa aktif dan NIM terbit
- Kurikulum, mata kuliah, dan prasyarat
- Jadwal kelas dan deteksi bentrok
- KRS dengan persetujuan dosen wali
- Presensi dan nilai, KHS, transkrip
- Tagihan UKT, cicilan, dan status pembayaran (pembayaran disimulasikan)
- Validator dan ekspor data format PDDikti
- Audit log

**Di luar ruang lingkup**

- LMS / e-learning (setara EdLink)
- Integrasi langsung ke Feeder PDDikti dan payment gateway sungguhan
- Akreditasi, tracer study, persuratan
- Aplikasi mobile native

**Metrik keberhasilan (target demo)**

| Metrik | Target |
| --- | --- |
| Endpoint KRS submit, p95 latency di 500 request bersamaan | < 300 ms |
| Overbooking kuota kelas saat load test KRS | 0 kasus |
| Aturan bisnis kritis yang punya unit test | 100% |
| Error validasi PDDikti terdeteksi sebelum ekspor | Semua aturan di bagian 5 |
| Waktu operator memproses satu KRS (persetujuan wali) | < 30 detik |

## 3. Stakeholder dan peran pengguna

Sistem memakai RBAC per tenant: satu akun bisa punya lebih dari satu peran (misalnya dosen yang juga kaprodi), dan semua akses dibatasi ke kampus serta prodi yang menjadi wewenangnya.

| Peran | Tanggung jawab utama | Akses kunci |
| --- | --- | --- |
| Super Admin (vendor) | Mengelola tenant kampus, paket, dan konfigurasi global | Semua tenant, tanpa akses nilai mahasiswa |
| Admin Kampus | Konfigurasi kampus, periode akademik, pengguna | Seluruh data satu kampus |
| Operator Akademik (BAAK) | Master data, jadwal, KRS, nilai, PDDikti | Data akademik seluruh prodi |
| Kaprodi | Kurikulum, plotting dosen, approval perubahan nilai | Prodinya sendiri |
| Dosen Wali | Menyetujui atau menolak KRS mahasiswa bimbingan | Mahasiswa bimbingannya |
| Dosen Pengampu | Presensi, input nilai kelas yang diajar | Kelas yang diajar |
| Bagian Keuangan | Tagihan, cicilan, dispensasi, rekonsiliasi | Data keuangan, status akademik read-only |
| Mahasiswa | KRS, lihat jadwal, KHS, transkrip, tagihan | Data dirinya sendiri |
| Calon Mahasiswa | Daftar PMB, unggah berkas, bayar formulir | Data pendaftarannya |

## 4. Modul inti dan kebutuhan fungsional

Sepuluh kelompok kebutuhan mengikuti urutan siklus mahasiswa dalam satu semester: PMB → registrasi dan tagihan → KRS → perkuliahan → nilai → pelaporan. Prioritas: **P0** wajib di MVP, **P1** jika waktu cukup.

### 4.1 Tenant, autentikasi, dan RBAC

- **FR-01 (P0)** Login dengan email/NIM/NIDN + password, JWT access token (15 menit) dan refresh token (rotasi).
- **FR-02 (P0)** Setiap request membawa tenant\_id dari token; query lintas tenant ditolak di layer repository.
- **FR-03 (P0)** Peran ganda per pengguna dan switch role di UI.
- **FR-04 (P1)** Reset password via email dan lock akun setelah 5 kali gagal login.

### 4.2 Master data

- **FR-05 (P0)** CRUD fakultas, prodi (jenjang D3/S1/S2), dosen, mahasiswa, ruang, dan periode akademik (Ganjil, Genap, Pendek).
- **FR-06 (P0)** Hanya satu periode berstatus aktif per tenant; periode punya jadwal tahapan (KRS, perkuliahan, UTS, UAS, input nilai).
- **FR-07 (P1)** Import mahasiswa dan dosen dari CSV dengan laporan baris gagal.

### 4.3 PMB (penerimaan mahasiswa baru)

- **FR-08 (P0)** Gelombang pendaftaran dengan kuota dan biaya formulir per gelombang.
- **FR-09 (P0)** Alur status pendaftar: Daftar → Bayar formulir → Lengkapi berkas → Verifikasi → Lulus seleksi / Tidak lulus → Daftar ulang.
- **FR-10 (P0)** Daftar ulang menerbitkan NIM otomatis dengan format yang bisa dikonfigurasi (misalnya tahun + kode prodi + nomor urut) tanpa duplikat.

### 4.4 Kurikulum dan mata kuliah

- **FR-11 (P0)** Kurikulum per prodi per angkatan, berisi mata kuliah, SKS, semester anjuran, dan sifat (wajib/pilihan).
- **FR-12 (P0)** Prasyarat mata kuliah: sudah lulus, atau lulus dengan nilai minimal tertentu.
- **FR-13 (P1)** Mapping ekuivalensi mata kuliah antar kurikulum.

### 4.5 Jadwal dan kelas

- **FR-14 (P0)** Kelas per mata kuliah per periode, dengan dosen pengampu, ruang, slot waktu, dan kuota.
- **FR-15 (P0)** Deteksi bentrok saat menyimpan jadwal: ruang sama di slot sama, atau dosen sama di slot sama.

### 4.6 KRS

- **FR-16 (P0)** Mahasiswa memilih kelas hanya di jendela waktu KRS dan hanya jika status aktif serta tagihan wajib lunas.
- **FR-17 (P0)** Batas SKS ditentukan IPS semester sebelumnya (contoh default: IPS ≥ 3,00 → 24 SKS; 2,50–2,99 → 21; 2,00–2,49 → 18; < 2,00 → 15), dapat dikonfigurasi per kampus.
- **FR-18 (P0)** Validasi prasyarat, bentrok jadwal pribadi, dan kuota kelas saat submit.
- **FR-19 (P0)** Status KRS: Draft → Diajukan → Disetujui / Ditolak (dengan catatan) oleh dosen wali.
- **FR-20 (P1)** Masa revisi KRS (tambah/batal) dengan jendela waktu sendiri.

### 4.7 Perkuliahan, presensi, dan nilai

- **FR-21 (P0)** Dosen membuka pertemuan dan mencatat presensi (Hadir, Izin, Sakit, Alpa).
- **FR-22 (P0)** Komponen nilai berbobot per kelas (contoh: tugas 20%, UTS 30%, UAS 50%); nilai angka dikonversi ke huruf dan bobot sesuai skala kampus.
- **FR-23 (P0)** Nilai dikunci (finalized) setelah batas input nilai; KHS, IPS, dan IPK dihitung dari nilai final.
- **FR-24 (P0)** Transkrip memakai nilai terbaik bila mata kuliah diulang.

### 4.8 Keuangan

- **FR-25 (P0)** Tagihan per mahasiswa per periode dibuat dari tarif prodi dan angkatan (UKT/SPP, praktikum, dll.).
- **FR-26 (P0)** Pembayaran disimulasikan lewat virtual account palsu dan webhook; status Belum bayar → Sebagian → Lunas.
- **FR-27 (P0)** Cicilan dan dispensasi yang disetujui keuangan membuka akses KRS walau belum lunas.
- **FR-28 (P1)** Laporan piutang per prodi dan per angkatan.

### 4.9 Pelaporan PDDikti

- **FR-29 (P0)** Validator pra-ekspor yang menjalankan aturan konsistensi data (bagian 5) dan menampilkan skor kesiapan per periode.
- **FR-30 (P0)** Ekspor data mahasiswa, kelas, KRS, nilai, dan aktivitas kuliah ke format CSV yang meniru struktur Feeder.
- **FR-31 (P0)** Setiap error menunjuk ke record dan halaman perbaikannya.

### 4.10 Lintas modul

- **FR-32 (P0)** Audit log untuk semua perubahan data akademik dan keuangan: siapa, kapan, nilai lama, nilai baru.
- **FR-33 (P1)** Notifikasi in-app (KRS disetujui, tagihan jatuh tempo, nilai terbit).

## 5. Edge case unik dan aturan bisnis kompleks

Bagian ini adalah inti portofolio: setiap edge case punya aturan yang jelas, penanganan teknis, dan skenario uji otomatis. Diagram status mahasiswa di akhir bagian ini menjadi acuan untuk semua aturan yang bergantung status.

### 5.1 Konkurensi dan integritas transaksi

| ID | Skenario | Aturan dan penanganan | Cara uji |
| --- | --- | --- | --- |
| EC-01 | 300 mahasiswa merebut 1 kursi terakhir kelas favorit di detik pertama KRS dibuka | Update kuota atomik: `UPDATE kelas SET terisi = terisi + 1 WHERE id = ? AND terisi < kuota`; 0 baris terdampak = kelas penuh. Insert KRS dan update kuota dalam satu transaksi | Load test k6, 500 request bersamaan, assert terisi ≤ kuota |
| EC-02 | Mahasiswa klik submit dua kali atau jaringan retry | Header `Idempotency-Key` disimpan 24 jam; request kedua mengembalikan respons pertama | Integration test kirim request sama 2 kali |
| EC-03 | Webhook pembayaran datang dua kali atau tidak berurutan | Dedup berdasarkan `transaction_id`; status tagihan dihitung ulang dari total pembayaran, bukan dari event terakhir | Kirim event duplikat dan terbalik, saldo harus sama |
| EC-04 | Dua operator daftar ulang bersamaan, NIM bisa kembar | Sequence per (prodi, angkatan) dengan row lock; unique index pada NIM per tenant | 50 goroutine daftar ulang paralel, NIM harus unik |

### 5.2 Aturan akademik

| ID | Skenario | Aturan dan penanganan | Cara uji |
| --- | --- | --- | --- |
| EC-05 | Prasyarat dipenuhi lewat mata kuliah kurikulum lama yang sudah diganti | Cek prasyarat lewat tabel ekuivalensi, bukan hanya kode mata kuliah | Unit test mahasiswa angkatan lama |
| EC-06 | Mahasiswa mengulang mata kuliah | KRS mata kuliah ulang tetap dihitung SKS; transkrip dan IPK memakai nilai terbaik, KHS menampilkan nilai semester itu | Unit test hitung IPK dengan nilai ulang |
| EC-07 | Semester pendek | Batas maksimal 9 SKS, tidak memakai aturan IPS, tidak menambah hitungan masa studi semester reguler | Unit test periode tipe Pendek |
| EC-08 | Mahasiswa cuti akademik | Tidak bisa KRS, tagihan diganti biaya cuti, maksimal 2 semester selama studi (dapat dikonfigurasi), wajib aktivasi sebelum KRS berikutnya | Unit test transisi status dan blokir KRS |
| EC-09 | Mendekati batas masa studi (contoh S1: 14 semester) | Peringatan di dashboard kaprodi pada sisa 2 semester; masuk daftar kandidat DO saat terlampaui, keputusan tetap manual | Unit test hitung semester aktif + cuti |
| EC-10 | IPK dihitung dengan float | Bobot disimpan sebagai integer (×100), perhitungan memakai decimal; pembulatan 2 desimal hanya saat tampil | Property test: hasil IPK sama di backend dan frontend |

### 5.3 Perubahan di tengah proses

| ID | Skenario | Aturan dan penanganan | Cara uji |
| --- | --- | --- | --- |
| EC-11 | Nilai salah setelah dikunci | Dosen mengajukan koreksi dengan alasan dan bukti; kaprodi approve; IPS/IPK/transkrip dihitung ulang; jika periode sudah diekspor, record ditandai perlu lapor ulang | E2E test alur koreksi + audit log |
| EC-12 | Dosen pengampu diganti di tengah semester | Presensi lama tetap atas nama dosen lama; hak input nilai pindah ke dosen baru sejak tanggal efektif | Integration test hak akses per tanggal |
| EC-13 | Kelas dibatalkan karena peserta di bawah minimum | KRS terkait dibatalkan otomatis, SKS dikembalikan, mahasiswa dinotifikasi dan diberi jendela pilih kelas pengganti | Integration test pembatalan kelas |
| EC-14 | KRS diajukan sebelum jendela tutup, tetapi dosen wali belum memproses | KRS berstatus Diajukan tetap bisa diproses sampai H+3 setelah jendela tutup (grace period) | Unit test dengan waktu disimulasikan |
| EC-15 | Pembayaran lunas masuk setelah KRS ditutup | Tidak membuka KRS otomatis; operator bisa memberi dispensasi KRS susulan yang tercatat di audit log | Integration test |

### 5.4 Multi-tenant dan waktu

| ID | Skenario | Aturan dan penanganan | Cara uji |
| --- | --- | --- | --- |
| EC-16 | Dua kampus memakai kode prodi atau NIM yang sama | Semua unique index menyertakan `tenant_id`; tidak ada lookup tanpa tenant | Test akses silang tenant harus 404 |
| EC-17 | Kampus di WITA atau WIT | Timezone disimpan per tenant; jendela KRS dan batas input nilai dievaluasi di zona waktu kampus, data disimpan UTC | Unit test jendela KRS lintas zona waktu |

### 5.5 Aturan validator PDDikti

Validator menjalankan aturan berikut sebelum ekspor; setiap pelanggaran punya tingkat Error (blokir ekspor) atau Peringatan.

| ID | Aturan | Tingkat |
| --- | --- | --- |
| PV-01 | Mahasiswa berstatus aktif di periode ini tapi tidak punya KRS | Error |
| PV-02 | Nilai ada tanpa KRS yang sesuai | Error |
| PV-03 | Kelas tanpa dosen pengampu | Error |
| PV-04 | SKS kelas berbeda dengan SKS mata kuliah di kurikulum | Error |
| PV-05 | NIK mahasiswa kosong, bukan 16 digit, atau duplikat | Error |
| PV-06 | Total SKS semester melebihi batas tanpa catatan dispensasi | Peringatan |
| PV-07 | Mahasiswa cuti atau non-aktif tetapi punya KRS | Error |
| PV-08 | Record ditandai perlu lapor ulang setelah koreksi nilai | Peringatan |

### 5.6 Status mahasiswa

&#91;embedded content: status mahasiswa · 6 status, transisi utama\]

Setiap perubahan status dicatat di riwayat\_status per periode; aturan KRS, tagihan, dan validator PDDikti membaca status dari riwayat ini, bukan dari satu kolom yang bisa ditimpa.

## 6. Kebutuhan non-fungsional

Target dirancang agar bisa dibuktikan saat demo, bukan sekadar klaim.

| Area | Kebutuhan |
| --- | --- |
| Performa | Endpoint baca p95 < 200 ms; submit KRS p95 < 300 ms pada 500 request bersamaan |
| Skalabilitas | Backend stateless sehingga bisa diskalakan horizontal; cache jadwal dan kuota di Redis |
| Keamanan | Password bcrypt, JWT dengan refresh rotation, rate limit login dan KRS per pengguna, validasi input di semua endpoint, CORS ketat |
| Isolasi data | Setiap query wajib tenant\_id (dicek di repository + test); opsi Row Level Security PostgreSQL sebagai lapis kedua |
| Auditabilitas | Audit log append-only untuk data akademik dan keuangan |
| Ketersediaan | Health check `/healthz` dan `/readyz`; migrasi database tanpa downtime (expand → migrate → contract) |
| Observability | Log terstruktur JSON dengan request\_id dan tenant\_id; metrik Prometheus (latency, error rate, kuota KRS) |
| Usability | Responsif sampai lebar 360 px untuk mahasiswa; operator bisa menyelesaikan tugas harian tanpa pelatihan panjang |
| Kualitas kode | Coverage aturan bisnis ≥ 80%; lint (golangci-lint, ESLint) dan test jalan otomatis di CI |

## 7. Arsitektur dan tech stack

Arsitekturnya modular monolith: satu binary Go Fiber yang dibagi per modul domain (akademik, KRS, keuangan, PMB, pelaporan), sehingga mudah dikembangkan sendirian namun batas modulnya tetap jelas jika kelak dipecah.

&#91;embedded content: arsitektur KampusOps · SPA, API, worker, data\]

Semua aturan dari bagian 5 hidup di layer service; handler dan worker hanya memanggilnya, sehingga aturan diuji sekali dan berlaku di semua jalur.

| Lapisan | Pilihan | Alasan |
| --- | --- | --- |
| Backend | Go 1.23, Fiber v2 | Performa tinggi untuk lonjakan trafik KRS |
| Akses data | pgx + sqlc, migrasi golang-migrate | SQL eksplisit, aman tipe, mudah kontrol locking |
| Database | PostgreSQL 16 | Transaksi, row lock, Row Level Security |
| Cache dan antrian | Redis + Asynq | Idempotency key, cache jadwal, job latar belakang |
| Frontend | Vue 3 (Composition API) + TypeScript, Vite | Standar SPA modern |
| State | Pinia, satu store per modul (auth, krs, nilai, tagihan) | State terpisah per domain, mudah dites |
| UI | Tailwind CSS + komponen Headless UI | Cepat dan konsisten |
| Testing | Go test + testcontainers, Vitest, Playwright, k6 | Unit, integrasi, E2E, dan load test |
| Infra | Docker Compose lokal, GitHub Actions CI, deploy ke VPS/Fly.io | Demo bisa diakses publik |
| Dokumentasi API | OpenAPI (swag) | Reviewer bisa mencoba endpoint |

**Struktur folder backend (ringkas)**

```
cmd/api, cmd/worker
internal/
  platform/   (config, db, redis, logger, middleware)
  academic/   (handler, service, repository)
  krs/
  finance/
  admission/
  reporting/  (validator PDDikti, ekspor)
  audit/
migrations/
```

## 8. Model data utama

Semua tabel bisnis membawa `tenant_id`, `created_at`, `updated_at`, dan semua unique index diawali `tenant_id`. Daftar ini adalah entitas inti; kolom detail ditentukan saat desain skema.

| Entitas | Kolom kunci | Relasi dan catatan |
| --- | --- | --- |
| tenant | id, nama, kode, timezone, konfigurasi\_aturan (JSONB) | Batas SKS, skala nilai, batas cuti disimpan per kampus |
| user, user\_role | id, email, password\_hash; role, scope\_prodi\_id | Satu user banyak peran |
| fakultas, prodi | kode, nama, jenjang, masa\_studi\_maks | Prodi milik fakultas |
| periode | tahun, tipe (Ganjil/Genap/Pendek), status, jadwal\_tahapan | Hanya satu aktif per tenant |
| mahasiswa | nim, nik, prodi\_id, angkatan, kurikulum\_id, dosen\_wali\_id, status | Status mengikuti diagram status mahasiswa |
| riwayat\_status | mahasiswa\_id, periode\_id, status, alasan | Dasar hitung masa studi dan cuti |
| dosen | nidn, nama, prodi\_id |  |
| kurikulum, mata\_kuliah | kode, nama, sks, semester, sifat | Mata kuliah milik kurikulum |
| prasyarat, ekuivalensi | mk\_id, mk\_syarat\_id, nilai\_min; mk\_lama\_id, mk\_baru\_id | Dipakai EC-05 |
| kelas | mk\_id, periode\_id, dosen\_id, ruang\_id, slot, kuota, terisi, status | `terisi` diupdate atomik (EC-01) |
| krs, krs\_detail | mahasiswa\_id, periode\_id, status, total\_sks; kelas\_id | Unique (mahasiswa, periode) |
| pertemuan, presensi | kelas\_id, tanggal, dosen\_id; mahasiswa\_id, status | Dosen dicatat per pertemuan (EC-12) |
| nilai, komponen\_nilai | krs\_detail\_id, angka, huruf, bobot\_x100, final\_at | Bobot integer (EC-10) |
| koreksi\_nilai | nilai\_id, nilai\_lama, nilai\_baru, alasan, status, approved\_by | EC-11 |
| tagihan, pembayaran | mahasiswa\_id, periode\_id, jenis, jumlah, status; transaction\_id (unique) | EC-03 |
| dispensasi | mahasiswa\_id, periode\_id, jenis, disetujui\_oleh | Buka KRS walau belum lunas |
| pendaftar\_pmb | gelombang\_id, status, berkas, prodi\_pilihan | Menjadi mahasiswa saat daftar ulang |
| audit\_log | actor\_id, entitas, entitas\_id, aksi, before, after (JSONB) | Append-only |
| idempotency\_key | key, user\_id, response, expires\_at | EC-02 |
| validasi\_pddikti | periode\_id, aturan, entitas\_id, tingkat, resolved\_at | Hasil validator |

## 9. Roadmap, asumsi, risiko, dan kriteria penerimaan

MVP dikerjakan dalam empat fase selama sekitar 8 minggu paruh waktu; setiap fase baru dianggap selesai jika gate-nya terbukti lewat test atau demo.

&#91;embedded content: roadmap · 4 fase · sekitar 8 minggu\]

Edge case konkurensi (EC-01 sampai EC-04) dikerjakan di fase 1–2 karena paling menentukan desain database.

**Asumsi**

- Data memakai seed sintetis: 2 kampus, 6 prodi, 1.500 mahasiswa, 2 periode berjalan.
- Format ekspor PDDikti meniru struktur kolom Feeder secara umum, bukan integrasi resmi.
- Pembayaran memakai simulator dengan webhook, bukan payment gateway sungguhan.
- Aturan akademik default (batas SKS, skala nilai, batas cuti) dapat diubah per tenant.

**Risiko**

| Risiko | Dampak | Mitigasi |
| --- | --- | --- |
| Ruang lingkup terlalu lebar untuk dikerjakan sendiri | MVP tidak selesai | Kerjakan semua P0 dulu; P1 hanya setelah gate fase 4 |
| Aturan akademik tiap kampus berbeda | Logika jadi kaku | Simpan aturan di konfigurasi tenant, bukan hardcode |
| Load test tidak representatif di laptop | Klaim performa lemah | Jalankan k6 di server deploy dan cantumkan hasilnya di README |
| Waktu habis di UI | Sisi backend kurang tergali | Pakai komponen UI siap pakai; fokus pada alur operator dan mahasiswa |

**Kriteria penerimaan MVP**

- [ ] Demo publik bisa login sebagai mahasiswa, dosen wali, dosen, operator, dan keuangan dengan akun contoh
- [ ] Satu semester penuh bisa dijalankan end-to-end: daftar ulang → tagihan → KRS → presensi → nilai → KHS → ekspor PDDikti
- [ ] Load test KRS 500 request bersamaan: p95 < 300 ms, 0 overbooking, hasil dicantumkan di README
- [ ] Semua edge case EC-01 sampai EC-17 punya test otomatis yang lulus di CI
- [ ] Validator PDDikti mendeteksi semua aturan PV-01 sampai PV-08 pada data seed yang sengaja dibuat salah
- [ ] README menjelaskan masalah bisnis, arsitektur, keputusan teknis, dan cara menjalankan dengan satu perintah `docker compose up`
- [ ] Video demo maksimal 3 menit
