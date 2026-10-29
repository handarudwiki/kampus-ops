package academic

import (
	"encoding/json"

	"github.com/kampusops/backend/internal/domain"
)

type Fakultas struct {
	ID   string `json:"id"`
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

type Prodi struct {
	ID            string `json:"id"`
	FakultasID    string `json:"fakultas_id"`
	FakultasNama  string `json:"fakultas_nama"`
	Kode          string `json:"kode"`
	Nama          string `json:"nama"`
	Jenjang       string `json:"jenjang"`
	MasaStudiMaks int    `json:"masa_studi_maks"`
}

type Dosen struct {
	ID        string `json:"id"`
	NIDN      string `json:"nidn"`
	Nama      string `json:"nama"`
	Email     string `json:"email"`
	ProdiID   string `json:"prodi_id"`
	ProdiNama string `json:"prodi_nama"`
}

type Ruang struct {
	ID        string `json:"id"`
	Kode      string `json:"kode"`
	Nama      string `json:"nama"`
	Kapasitas int    `json:"kapasitas"`
}

type Periode struct {
	ID      string         `json:"id"`
	Kode    string         `json:"kode"`
	Tahun   int            `json:"tahun"`
	Tipe    string         `json:"tipe"`
	Status  string         `json:"status"`
	Tahapan domain.Tahapan `json:"tahapan"`
	// ExportedAt is internal (EC-11 "perlu lapor ulang").
	ExportedAt *string `json:"-"`
}

type Mahasiswa struct {
	ID            string  `json:"id"`
	NIM           string  `json:"nim"`
	NIK           *string `json:"nik"`
	Nama          string  `json:"nama"`
	Email         string  `json:"email"`
	ProdiID       string  `json:"prodi_id"`
	ProdiNama     string  `json:"prodi_nama"`
	Angkatan      int     `json:"angkatan"`
	KurikulumID   string  `json:"kurikulum_id"`
	DosenWaliID   *string `json:"dosen_wali_id"`
	DosenWaliNama *string `json:"dosen_wali_nama"`
	Status        string  `json:"status"`
	IPK           string  `json:"ipk"`
	SksLulus      int     `json:"sks_lulus"`
	SemesterKe    int     `json:"semester_ke"`
	// internal
	UserID        *string `json:"-"`
	MasaStudiMaks int     `json:"-"`
}

type RiwayatStatus struct {
	PeriodeKode string  `json:"periode_kode"`
	Status      string  `json:"status"`
	Alasan      *string `json:"alasan"`
	CreatedAt   string  `json:"created_at"`
}

type Kurikulum struct {
	ID            string `json:"id"`
	ProdiID       string `json:"prodi_id"`
	Nama          string `json:"nama"`
	AngkatanMulai int    `json:"angkatan_mulai"`
	Aktif         bool   `json:"aktif"`
}

type PrasyaratItem struct {
	MkID     string  `json:"mk_id"`
	Kode     string  `json:"kode"`
	Nama     string  `json:"nama"`
	NilaiMin *string `json:"nilai_min"`
}

type MataKuliah struct {
	ID          string          `json:"id"`
	KurikulumID string          `json:"kurikulum_id"`
	Kode        string          `json:"kode"`
	Nama        string          `json:"nama"`
	Sks         int             `json:"sks"`
	Semester    int             `json:"semester"`
	Sifat       string          `json:"sifat"`
	Prasyarat   []PrasyaratItem `json:"prasyarat"`
}

type Ekuivalensi struct {
	ID         string `json:"id"`
	MkLamaID   string `json:"mk_lama_id"`
	MkLamaKode string `json:"mk_lama_kode"`
	MkLamaNama string `json:"mk_lama_nama"`
	MkBaruID   string `json:"mk_baru_id"`
	MkBaruKode string `json:"mk_baru_kode"`
	MkBaruNama string `json:"mk_baru_nama"`
}

type Kelas struct {
	ID        string  `json:"id"`
	PeriodeID string  `json:"periode_id"`
	MkID      string  `json:"mk_id"`
	MkKode    string  `json:"mk_kode"`
	MkNama    string  `json:"mk_nama"`
	Sks       int     `json:"sks"`
	Nama      string  `json:"nama"`
	DosenID   *string `json:"dosen_id"`
	DosenNama *string `json:"dosen_nama"`
	RuangID   *string `json:"ruang_id"`
	RuangNama *string `json:"ruang_nama"`
	Slot      Slot    `json:"slot"`
	Kuota     int     `json:"kuota"`
	Terisi    int     `json:"terisi"`
	Status    string  `json:"status"`
}

func (k Kelas) Scheduled() ScheduledClass {
	return ScheduledClass{ID: k.ID, RuangID: k.RuangID, DosenID: k.DosenID, Slot: k.Slot}
}

type KhsRow struct {
	MkKode    string  `json:"mk_kode"`
	MkNama    string  `json:"mk_nama"`
	Sks       int     `json:"sks"`
	Huruf     *string `json:"huruf"`
	BobotX100 *int    `json:"bobot_x100"`
	IsUlang   bool    `json:"is_ulang"`
}

type Khs struct {
	Periode  Periode  `json:"periode"`
	Rows     []KhsRow `json:"rows"`
	TotalSks int      `json:"total_sks"`
	IPS      string   `json:"ips"`
	IPK      string   `json:"ipk"`
}

type TranskripRow struct {
	KhsRow
	Semester    int    `json:"semester"`
	PeriodeKode string `json:"periode_kode"`
}

type Transkrip struct {
	Mahasiswa Mahasiswa      `json:"mahasiswa"`
	Rows      []TranskripRow `json:"rows"`
	TotalSks  int            `json:"total_sks"`
	IPK       string         `json:"ipk"`
}

func parseTahapan(b []byte) domain.Tahapan {
	var t domain.Tahapan
	_ = json.Unmarshal(b, &t)
	return t
}
