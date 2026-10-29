DROP TABLE IF EXISTS validasi_pddikti, validasi_run, notifikasi, audit_log, dispensasi, pembayaran, cicilan,
  tagihan, pendaftar, gelombang, tarif, koreksi_nilai, nilai, presensi, pertemuan, krs_detail, krs,
  komponen_nilai, kelas_dosen, kelas, nim_sequence, riwayat_status, mahasiswa, ekuivalensi, prasyarat,
  mata_kuliah, kurikulum, periode, dosen, ruang, refresh_token, user_role, prodi, fakultas, users, tenant CASCADE;
DROP FUNCTION IF EXISTS audit_log_append_only();
DROP FUNCTION IF EXISTS iso(timestamptz);
