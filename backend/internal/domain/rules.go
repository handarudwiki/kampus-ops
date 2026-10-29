// Package domain contains tenant-configurable academic rules and pure calculation
// helpers shared by several modules (grading, IPS/IPK, student status, time).
// Nothing here touches the database, so everything is unit-testable.
package domain

import (
	"encoding/json"
	"sort"
)

// SKSRule maps a minimum previous IPS (×100) to the max SKS allowed (FR-17).
type SKSRule struct {
	MinIPSx100 int `json:"min_ips_x100"`
	MaxSKS     int `json:"max_sks"`
}

// GradeBand maps a minimum numeric score (×100) to a letter and bobot (×100).
type GradeBand struct {
	Huruf       string `json:"huruf"`
	MinAngkax100 int   `json:"min_angka_x100"`
	BobotX100   int    `json:"bobot_x100"`
}

// Komponen is a default grade component definition (FR-22).
type Komponen struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	BobotPersen int    `json:"bobot_persen"`
}

// Rules is tenant.konfigurasi_aturan (JSONB). Missing fields fall back to defaults.
type Rules struct {
	SKSRules             []SKSRule   `json:"sks_rules"`
	SKSDefaultBaru       int         `json:"sks_default_baru"`
	SKSPendekMax         int         `json:"sks_pendek_max"`
	GradeScale           []GradeBand `json:"grade_scale"`
	LulusMinBobotX100    int         `json:"lulus_min_bobot_x100"`
	CutiMax              int         `json:"cuti_max"`
	MasaStudiWarningSisa int         `json:"masa_studi_warning_sisa"`
	KRSGraceDays         int         `json:"krs_grace_days"`
	NIMFormat            string      `json:"nim_format"`
	MinPesertaKelas      int         `json:"min_peserta_kelas"`
	BiayaCuti            int64       `json:"biaya_cuti"`
	VAPrefix             string      `json:"va_prefix"`
	KomponenDefault      []Komponen  `json:"komponen_default"`
}

// DefaultRules are the BRD defaults (FR-17, EC-07, EC-08, EC-14).
func DefaultRules() Rules {
	return Rules{
		SKSRules: []SKSRule{
			{MinIPSx100: 300, MaxSKS: 24},
			{MinIPSx100: 250, MaxSKS: 21},
			{MinIPSx100: 200, MaxSKS: 18},
			{MinIPSx100: 0, MaxSKS: 15},
		},
		SKSDefaultBaru: 20,
		SKSPendekMax:   9,
		GradeScale: []GradeBand{
			{"A", 8500, 400}, {"A-", 8000, 370}, {"B+", 7500, 330}, {"B", 7000, 300},
			{"B-", 6500, 270}, {"C+", 6000, 230}, {"C", 5500, 200}, {"D", 4000, 100}, {"E", 0, 0},
		},
		LulusMinBobotX100:    100,
		CutiMax:              2,
		MasaStudiWarningSisa: 2,
		KRSGraceDays:         3,
		NIMFormat:            "{angkatan}{prodi}{seq:4}",
		MinPesertaKelas:      5,
		BiayaCuti:            500000,
		VAPrefix:             "8808",
		KomponenDefault: []Komponen{
			{"tugas", "Tugas", 20}, {"uts", "UTS", 30}, {"uas", "UAS", 50},
		},
	}
}

// ParseRules merges a tenant JSON config over the defaults.
func ParseRules(raw []byte) Rules {
	r := DefaultRules()
	if len(raw) > 0 {
		var o Rules
		if err := json.Unmarshal(raw, &o); err == nil {
			if len(o.SKSRules) > 0 {
				r.SKSRules = o.SKSRules
			}
			if o.SKSDefaultBaru > 0 {
				r.SKSDefaultBaru = o.SKSDefaultBaru
			}
			if o.SKSPendekMax > 0 {
				r.SKSPendekMax = o.SKSPendekMax
			}
			if len(o.GradeScale) > 0 {
				r.GradeScale = o.GradeScale
			}
			if o.LulusMinBobotX100 > 0 {
				r.LulusMinBobotX100 = o.LulusMinBobotX100
			}
			if o.CutiMax > 0 {
				r.CutiMax = o.CutiMax
			}
			if o.MasaStudiWarningSisa > 0 {
				r.MasaStudiWarningSisa = o.MasaStudiWarningSisa
			}
			if o.KRSGraceDays > 0 {
				r.KRSGraceDays = o.KRSGraceDays
			}
			if o.NIMFormat != "" {
				r.NIMFormat = o.NIMFormat
			}
			if o.MinPesertaKelas > 0 {
				r.MinPesertaKelas = o.MinPesertaKelas
			}
			if o.BiayaCuti > 0 {
				r.BiayaCuti = o.BiayaCuti
			}
			if o.VAPrefix != "" {
				r.VAPrefix = o.VAPrefix
			}
			if len(o.KomponenDefault) > 0 {
				r.KomponenDefault = o.KomponenDefault
			}
		}
	}
	sort.SliceStable(r.SKSRules, func(i, j int) bool { return r.SKSRules[i].MinIPSx100 > r.SKSRules[j].MinIPSx100 })
	sort.SliceStable(r.GradeScale, func(i, j int) bool { return r.GradeScale[i].MinAngkax100 > r.GradeScale[j].MinAngkax100 })
	return r
}
