# KampusOps — Panduan fitur

Tur fitur per peran, mengikuti siklus satu semester di BRD: **PMB → registrasi & tagihan → KRS → perkuliahan → nilai → pelaporan PDDikti**.
Semua gambar ada di [`/screenshots`](../screenshots) dan bisa dibuat ulang dengan `cd frontend && npm run screenshots` (lihat [README](../README.md#screenshots)).

Akun demo (kampus `UNIKA`, kata sandi `password123`) bisa dipakai langsung dari tombol **Masuk cepat** di halaman login.

---

## 1. Masuk, peran, dan navigasi

| | |
|---|---|
| ![Login](../screenshots/01-login.png) | **Login (FR-01).** Email, NIM, atau NIDN plus pilihan kampus (tenant). Tombol akun demo langsung masuk sebagai peran tersebut. Lima kali gagal login mengunci akun sementara (FR-04). |
| ![Command palette](../screenshots/80-command-palette.png) | **Command palette (⌘K / Ctrl+K).** Lompat ke halaman mana pun milik peran aktif, ganti peran (FR-03), atau ganti tema. |

Satu akun bisa punya banyak peran. Contohnya NIDN `0011223344` adalah dosen, dosen wali, dan kaprodi sekaligus. Peran aktif dipilih di kartu kampus pada sidebar, dan setiap request membawa header `X-Active-Role`.

## 2. Mahasiswa

| | |
|---|---|
| ![Dashboard mahasiswa](../screenshots/10-mahasiswa-dashboard.png) | **Beranda.** Kalender semester menandai tahapan yang sedang berjalan (KRS, perkuliahan, UTS, UAS, input nilai) di zona waktu kampus (EC-17), ditambah KPI IPK, SKS, tagihan, dan status KRS. |
| ![KRS](../screenshots/11-mahasiswa-krs.png) | **KRS (FR-16..19).** Penawaran kelas menampilkan alasan sebuah kelas tidak bisa diambil (prasyarat, bentrok, penuh). Kuota tampil langsung. Batas SKS ditentukan IPS lalu (FR-17), dan ringkasan menampilkan cincin SKS serta alur Draft → Diajukan → Disetujui. Setiap aksi mengirim `Idempotency-Key` (EC-02), dan kuota diupdate atomik (EC-01). |
| ![KRS diblokir](../screenshots/17-mahasiswa-krs-diblokir.png) | **KRS diblokir.** Mahasiswa dengan tagihan belum lunas dan tanpa dispensasi melihat alasannya beserta tautan ke tagihan (FR-16, FR-27). |
| ![Jadwal](../screenshots/12-mahasiswa-jadwal.png) | **Jadwal kuliah.** Kelas dari KRS yang sudah disetujui, dikelompokkan per hari. |
| ![KHS](../screenshots/13-mahasiswa-khs.png) | **KHS (FR-23).** Nilai final per periode, IPS, dan IPK. Mata kuliah ulang ditandai (EC-06). |
| ![Transkrip](../screenshots/14-mahasiswa-transkrip.png) | **Transkrip (FR-24).** Dikelompokkan per semester dan memakai nilai terbaik untuk mata kuliah yang diulang. Bobot disimpan ×100 dan ditampilkan tanpa galat float (EC-10). |
| ![Tagihan](../screenshots/15-mahasiswa-tagihan.png) | **Tagihan (FR-25..27).** Progres pembayaran, nomor VA, jadwal cicilan, dan **simulasi bayar** lewat VA palsu. Webhook diproses server, dan status dihitung ulang dari total pembayaran (EC-03). |
| ![Profil](../screenshots/16-mahasiswa-profil.png) | **Profil.** Data induk dan riwayat status per periode, yang menjadi dasar hitung masa studi dan cuti. |

## 3. Dosen wali

| | |
|---|---|
| ![Dashboard wali](../screenshots/20-dosenwali-dashboard.png) | **Beranda wali.** Jumlah KRS menunggu dan mahasiswa bimbingan. |
| ![Perwalian](../screenshots/21-dosenwali-perwalian.png) | **Antrian persetujuan.** KRS bisa disetujui satu klik dari daftar (target BRD: kurang dari 30 detik per KRS). KRS yang sudah diajukan tetap bisa diproses sampai H+3 setelah jendela KRS ditutup (EC-14). |
| ![Detail KRS](../screenshots/22-dosenwali-krs-detail.png) | **Detail KRS.** SKS terhadap batas, IPS lalu, IPK, dan daftar kelas. Menolak KRS wajib disertai catatan untuk mahasiswa. |

## 4. Dosen pengampu

| | |
|---|---|
| ![Kelas saya](../screenshots/30-dosen-mengajar.png) | **Kelas saya.** Kelas yang diampu di periode aktif. |
| ![Presensi](../screenshots/31-dosen-presensi.png) | **Presensi (FR-21).** Dosen membuka pertemuan lalu menandai H/I/S/A per mahasiswa, dengan tombol "Semua hadir". Dosen dicatat per pertemuan, sehingga pergantian dosen tidak mengubah riwayat (EC-12). |
| ![Nilai](../screenshots/32-dosen-nilai.png) | **Nilai (FR-22, FR-23).** Komponen nilai berbobot dengan pratinjau angka yang dihitung secara integer, sama dengan backend (EC-10). Setelah **Kunci nilai**, perubahan hanya bisa lewat koreksi. |
| ![Koreksi dosen](../screenshots/33-dosen-koreksi.png) | **Koreksi nilai (EC-11).** Riwayat koreksi yang diajukan dosen. |

## 5. Kaprodi

| | |
|---|---|
| ![Dashboard kaprodi](../screenshots/40-kaprodi-dashboard.png) | **Beranda kaprodi.** Peringatan masa studi dan koreksi nilai yang menunggu. |
| ![Kurikulum](../screenshots/41-kaprodi-kurikulum.png) | **Kurikulum (FR-11..13).** Mata kuliah per semester, prasyarat dengan nilai minimal, dan tab ekuivalensi antar kurikulum (EC-05). |
| ![Koreksi kaprodi](../screenshots/42-kaprodi-koreksi-nilai.png) | **Persetujuan koreksi nilai (EC-11).** Menampilkan perubahan huruf dan angka lama ke baru. Persetujuan menghitung ulang IPS/IPK/transkrip dan menandai record "perlu lapor ulang" (PV-08). |
| ![Masa studi](../screenshots/43-kaprodi-masa-studi.png) | **Masa studi (EC-09).** Peringatan saat sisa masa studi 2 semester. Mahasiswa yang melewati batas menjadi kandidat DO, tetapi keputusannya tetap manual. |

## 6. Operator akademik (BAAK)

| | |
|---|---|
| ![Dashboard operator](../screenshots/50-operator-dashboard.png) | **Beranda operator.** Skor PDDikti, progres KRS, dan jumlah kelas. |
| ![Mahasiswa](../screenshots/51-operator-mahasiswa.png) | **Data induk (FR-05).** Semua layar master data (mahasiswa, dosen, prodi, fakultas, ruang, periode) dirender oleh satu `ResourcePage` berbasis konfigurasi, jadi perilakunya identik: pencarian, filter, paginasi, form modal, dan error validasi per field. |
| ![Detail mahasiswa](../screenshots/52-operator-mahasiswa-detail.png) | **Detail mahasiswa.** Riwayat status, ubah status (batas cuti, EC-08), dan perbaikan NIK langsung untuk PV-05. |
| ![Kelas](../screenshots/53-operator-kelas.png) | **Jadwal & kelas (FR-14).** Kuota, dosen, ruang, dan slot. Aksi tersedia untuk ganti dosen dengan tanggal efektif (EC-12) dan batalkan kelas, yang otomatis membatalkan KRS terkait (EC-13). |
| ![Form kelas](../screenshots/54-operator-kelas-form.png) | **Deteksi bentrok (FR-15).** Bentrok ruang atau dosen dikembalikan sebagai `409` dan kelas yang bentrok ditampilkan di dalam form. |
| ![PDDikti](../screenshots/55-operator-pddikti.png) | **Validator PDDikti (FR-29..31).** Skor kesiapan, jumlah temuan per aturan PV-01..PV-08 (bisa diklik sebagai filter), dan tombol **Perbaiki** yang membuka halaman record terkait. Ekspor CSV format Feeder diblokir selama masih ada error. |
| ![PMB](../screenshots/56-operator-pmb.png) | **PMB (FR-08..10).** Pipeline status pendaftar dengan jumlah per status dan transisi yang diizinkan. Daftar ulang menerbitkan NIM dari sequence per (prodi, angkatan) dengan row lock (EC-04). |
| ![Audit](../screenshots/57-operator-audit-log.png) | **Audit log (FR-32).** Catatan append-only. Klik satu baris untuk melihat diff sebelum/sesudah per kolom. |
| ![Dosen](../screenshots/58-operator-dosen.png) | **Dosen.** Contoh layar master data yang sama polanya. |
| ![Periode](../screenshots/59-operator-periode.png) | **Periode akademik (FR-06).** Jadwal tahapan, dengan hanya satu periode aktif per kampus. |

## 7. Bagian keuangan

| | |
|---|---|
| ![Dashboard keuangan](../screenshots/60-keuangan-dashboard.png) | **Beranda keuangan.** Piutang, persentase lunas, dan tren pembayaran. |
| ![Tagihan](../screenshots/61-keuangan-tagihan.png) | **Tagihan mahasiswa (FR-25, FR-26).** Buat tagihan periode dari tarif, gauge komposisi status, terbitkan VA, atur cicilan 2–4×, riwayat pembayaran, dan simulasi bayar. |
| ![Dispensasi](../screenshots/62-keuangan-dispensasi.png) | **Dispensasi (FR-27, EC-15).** Cicilan, penundaan, dan KRS susulan. Pelunasan setelah KRS ditutup tidak membuka KRS otomatis. |
| ![Piutang](../screenshots/63-keuangan-piutang.png) | **Piutang (FR-28).** Dikelompokkan per prodi atau angkatan. |

## 8. Admin kampus, vendor, dan calon mahasiswa

| | |
|---|---|
| ![Dashboard admin](../screenshots/70-admin-dashboard.png) | **Beranda admin kampus.** |
| ![Pengguna](../screenshots/71-admin-pengguna.png) | **Pengguna & peran (FR-03).** Banyak peran per akun, dengan peran kaprodi/dosen dibatasi ke prodi tertentu. |
| ![Tenant](../screenshots/72-superadmin-tenants.png) | **Tenant kampus.** Super admin vendor mengelola kampus dan paketnya, tanpa akses ke nilai mahasiswa. Dua kampus demo memakai kode prodi/NIM yang sama untuk membuktikan isolasi (EC-16). |
| ![Status PMB](../screenshots/73-calon-status-pmb.png) | **Calon mahasiswa.** Status seleksi dan kelengkapan berkas. |
| ![Daftar](../screenshots/02-pmb-daftar.png) | **Pendaftaran PMB publik.** |

## 9. Tema gelap, mobile, dan design system

| | | |
|---|---|---|
| ![Dark operator](../screenshots/81-dark-operator-dashboard.png) | ![Dark KRS](../screenshots/82-dark-mahasiswa-krs.png) | ![Design system](../screenshots/03-design-system.png) |
| Tema gelap mengikuti OS atau dipilih di sidebar. | KRS dalam tema gelap. | Showcase `/design-system`. |

| | | |
|---|---|---|
| ![Mobile dashboard](../screenshots/83-mobile-dashboard.png) | ![Mobile KRS](../screenshots/84-mobile-krs.png) | ![Mobile tagihan](../screenshots/85-mobile-tagihan.png) |

Responsif sampai lebar 360 px. Di mobile, navigasi utama pindah ke tab bar bawah, sidebar menjadi drawer, modal menjadi bottom sheet, dan kolom tabel yang kurang penting disembunyikan.

Detail token, komponen, dan aturan konsistensi UI ada di [DESIGN_SYSTEM.md](DESIGN_SYSTEM.md).
