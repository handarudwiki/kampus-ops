package academic

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/kampusops/backend/internal/audit"
	"github.com/kampusops/backend/internal/platform/appctx"
	"github.com/kampusops/backend/internal/platform/db"
	"github.com/kampusops/backend/internal/platform/httpx"
)

func (s *Service) ListKurikulum(ctx context.Context, prodiID string) ([]Kurikulum, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, prodi_id, nama, angkatan_mulai, aktif FROM kurikulum WHERE tenant_id=$1 AND ($2='' OR prodi_id::text=$2)
		ORDER BY prodi_id, angkatan_mulai DESC`, appctx.TenantID(ctx), prodiID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Kurikulum])
	return nonNil(out), err
}

// ListMataKuliah returns courses (with prerequisites) of a curriculum.
func (s *Service) ListMataKuliah(ctx context.Context, kurikulumID string) ([]MataKuliah, error) {
	return s.queryMK(ctx, `mk.kurikulum_id=$2`, kurikulumID)
}

func (s *Service) getMK(ctx context.Context, id string) (MataKuliah, error) {
	ms, err := s.queryMK(ctx, `mk.id=$2`, id)
	if err != nil {
		return MataKuliah{}, err
	}
	if len(ms) == 0 {
		return MataKuliah{}, httpx.NotFound("Mata kuliah tidak ditemukan")
	}
	return ms[0], nil
}

func (s *Service) queryMK(ctx context.Context, where string, arg string) ([]MataKuliah, error) {
	tid := appctx.TenantID(ctx)
	rows, err := s.pool.Query(ctx, `SELECT mk.id, mk.kurikulum_id, mk.kode, mk.nama, mk.sks, mk.semester, mk.sifat FROM mata_kuliah mk
		WHERE mk.tenant_id=$1 AND `+where+` ORDER BY mk.semester, mk.kode`, tid, arg)
	if err != nil {
		return nil, err
	}
	out := []MataKuliah{}
	idx := map[string]int{}
	for rows.Next() {
		var m MataKuliah
		if err := rows.Scan(&m.ID, &m.KurikulumID, &m.Kode, &m.Nama, &m.Sks, &m.Semester, &m.Sifat); err != nil {
			rows.Close()
			return nil, err
		}
		m.Prasyarat = []PrasyaratItem{}
		idx[m.ID] = len(out)
		out = append(out, m)
	}
	rows.Close()
	if len(out) == 0 {
		return out, nil
	}
	ids := make([]string, 0, len(out))
	for _, m := range out {
		ids = append(ids, m.ID)
	}
	prow, err := s.pool.Query(ctx, `SELECT pr.mk_id, s.id, s.kode, s.nama, pr.nilai_min FROM prasyarat pr JOIN mata_kuliah s ON s.id=pr.mk_syarat_id
		WHERE pr.tenant_id=$1 AND pr.mk_id = ANY($2::uuid[]) ORDER BY s.kode`, tid, ids)
	if err != nil {
		return nil, err
	}
	defer prow.Close()
	for prow.Next() {
		var mk string
		var p PrasyaratItem
		if err := prow.Scan(&mk, &p.MkID, &p.Kode, &p.Nama, &p.NilaiMin); err != nil {
			return nil, err
		}
		out[idx[mk]].Prasyarat = append(out[idx[mk]].Prasyarat, p)
	}
	return out, prow.Err()
}

type MataKuliahInput struct {
	KurikulumID string `json:"kurikulum_id"`
	Kode        string `json:"kode"`
	Nama        string `json:"nama"`
	Sks         int    `json:"sks"`
	Semester    int    `json:"semester"`
	Sifat       string `json:"sifat"`
	Prasyarat   []struct {
		MkID     string  `json:"mk_id"`
		NilaiMin *string `json:"nilai_min"`
	} `json:"prasyarat"`
}

func (s *Service) SaveMataKuliah(ctx context.Context, id string, in MataKuliahInput) (MataKuliah, error) {
	if in.Sifat == "" {
		in.Sifat = "wajib"
	}
	t, err := s.tenants.Current(ctx)
	if err != nil {
		return MataKuliah{}, err
	}
	var before any
	if id != "" {
		b, err := s.getMK(ctx, id)
		if err != nil {
			return MataKuliah{}, err
		}
		before = b
		if in.KurikulumID == "" {
			in.KurikulumID = b.KurikulumID
		}
		if err := checkKaprodiScope(ctx, s.pool, id); err != nil {
			return MataKuliah{}, err
		}
	}
	v := httpx.NewV().Required("kode", in.Kode).Required("nama", in.Nama).Range("sks", in.Sks, 1, 8).Range("semester", in.Semester, 1, 14).
		OneOf("sifat", in.Sifat, "wajib", "pilihan")
	for _, p := range in.Prasyarat {
		v.UUID("prasyarat", p.MkID)
		if p.NilaiMin != nil && *p.NilaiMin != "" {
			_, ok := t.Rules.BobotOf(*p.NilaiMin)
			v.Check(ok, "prasyarat.nilai_min", "huruf mutu tidak dikenal: "+*p.NilaiMin)
		}
		v.Check(p.MkID != id || id == "", "prasyarat", "mata kuliah tidak boleh menjadi prasyarat dirinya sendiri")
	}
	if err := v.Err(); err != nil {
		return MataKuliah{}, err
	}
	if err := MustExist(ctx, s.pool, "kurikulum", "kurikulum_id", in.KurikulumID); err != nil {
		return MataKuliah{}, err
	}
	for _, p := range in.Prasyarat {
		if err := MustExist(ctx, s.pool, "mata_kuliah", "prasyarat", p.MkID); err != nil {
			return MataKuliah{}, err
		}
	}
	err = db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		tid := appctx.TenantID(ctx)
		var err error
		if id == "" {
			err = tx.QueryRow(ctx, `INSERT INTO mata_kuliah (tenant_id, kurikulum_id, kode, nama, sks, semester, sifat) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
				tid, in.KurikulumID, in.Kode, in.Nama, in.Sks, in.Semester, in.Sifat).Scan(&id)
		} else {
			_, err = tx.Exec(ctx, `UPDATE mata_kuliah SET kurikulum_id=$3, kode=$4, nama=$5, sks=$6, semester=$7, sifat=$8, updated_at=now() WHERE tenant_id=$1 AND id=$2`,
				tid, id, in.KurikulumID, in.Kode, in.Nama, in.Sks, in.Semester, in.Sifat)
		}
		if err != nil {
			return dup(err, "kode")
		}
		if _, err := tx.Exec(ctx, `DELETE FROM prasyarat WHERE tenant_id=$1 AND mk_id=$2`, tid, id); err != nil {
			return err
		}
		for _, p := range in.Prasyarat {
			nm := p.NilaiMin
			if nm != nil && *nm == "" {
				nm = nil
			}
			if _, err := tx.Exec(ctx, `INSERT INTO prasyarat (tenant_id, mk_id, mk_syarat_id, nilai_min) VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`, tid, id, p.MkID, nm); err != nil {
				return err
			}
		}
		return audit.Write(ctx, tx, "mata_kuliah", id, saveAksi(before), before, in)
	})
	if err != nil {
		return MataKuliah{}, err
	}
	return s.getMK(ctx, id)
}

func (s *Service) ListEkuivalensi(ctx context.Context, kurikulumID string) ([]Ekuivalensi, error) {
	rows, err := s.pool.Query(ctx, `SELECT e.id, a.id, a.kode, a.nama, b.id, b.kode, b.nama FROM ekuivalensi e
		JOIN mata_kuliah a ON a.id=e.mk_lama_id JOIN mata_kuliah b ON b.id=e.mk_baru_id
		WHERE e.tenant_id=$1 AND ($2='' OR b.kurikulum_id::text=$2) ORDER BY b.kode`, appctx.TenantID(ctx), kurikulumID)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Ekuivalensi])
	return nonNil(out), err
}

func (s *Service) CreateEkuivalensi(ctx context.Context, lama, baru string) (Ekuivalensi, error) {
	if err := httpx.NewV().UUID("mk_lama_id", lama).UUID("mk_baru_id", baru).Check(lama != baru, "mk_baru_id", "harus berbeda dari mk_lama_id").Err(); err != nil {
		return Ekuivalensi{}, err
	}
	if err := MustExist(ctx, s.pool, "mata_kuliah", "mk_lama_id", lama); err != nil {
		return Ekuivalensi{}, err
	}
	if err := MustExist(ctx, s.pool, "mata_kuliah", "mk_baru_id", baru); err != nil {
		return Ekuivalensi{}, err
	}
	var id string
	err := db.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO ekuivalensi (tenant_id, mk_lama_id, mk_baru_id) VALUES ($1,$2,$3) RETURNING id`, appctx.TenantID(ctx), lama, baru).Scan(&id)
		if err != nil {
			return dup(err, "mk_baru_id")
		}
		return audit.Write(ctx, tx, "ekuivalensi", id, "create", nil, map[string]string{"mk_lama_id": lama, "mk_baru_id": baru})
	})
	if err != nil {
		return Ekuivalensi{}, err
	}
	all, err := s.ListEkuivalensi(ctx, "")
	if err != nil {
		return Ekuivalensi{}, err
	}
	for _, e := range all {
		if e.ID == id {
			return e, nil
		}
	}
	return Ekuivalensi{}, httpx.NotFound("")
}
