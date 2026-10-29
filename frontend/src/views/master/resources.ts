import { CalendarCheck2 } from 'lucide-vue-next'
import { api } from '@/api/client'
import { date, ip, statusOptions } from '@/design-system'
import { optionsFrom, type ResourceConfig } from './resource'

const prodiOpts = optionsFrom('/prodi', (r) => `${r.jenjang} ${r.nama}`)
const fakultasOpts = optionsFrom('/fakultas', (r) => r.nama as string)
const dosenOpts = optionsFrom('/dosen', (r) => r.nama as string)
const thisYear = new Date().getFullYear()
const angkatanOpts = Array.from({ length: 8 }, (_, i) => ({ value: thisYear - i, label: String(thisYear - i) }))

export const RESOURCES = {
  mahasiswa: {
    title: 'Mahasiswa',
    description: 'Data induk mahasiswa. Status dibaca dari riwayat status per periode.',
    endpoint: '/mahasiswa',
    singular: 'Mahasiswa',
    detail: (id) => `/mahasiswa/${id}`,
    canDelete: false,
    columns: [
      { key: 'nim', label: 'NIM', mono: true, width: '130px' },
      { key: 'nama', label: 'Nama' },
      { key: 'prodi_nama', label: 'Prodi', hideMobile: true },
      { key: 'angkatan', label: 'Angkatan', align: 'right', hideMobile: true },
      { key: 'ipk', label: 'IPK', align: 'right', hideMobile: true, format: (v) => ip(v as string) },
      { key: 'status', label: 'Status', status: 'mahasiswa' },
    ],
    filters: [
      { key: 'prodi_id', label: 'Semua prodi', options: prodiOpts },
      { key: 'angkatan', label: 'Semua angkatan', options: angkatanOpts },
      { key: 'status', label: 'Semua status', options: statusOptions('mahasiswa') },
    ],
    fields: [
      { key: 'nama', label: 'Nama lengkap', required: true, wide: true },
      { key: 'nim', label: 'NIM', hint: 'Kosongkan untuk terbit otomatis', createOnly: true },
      { key: 'nik', label: 'NIK', hint: '16 digit, wajib untuk PDDikti' },
      { key: 'email', label: 'Email', type: 'email', required: true },
      { key: 'prodi_id', label: 'Prodi', type: 'select', options: prodiOpts, required: true },
      { key: 'angkatan', label: 'Angkatan', type: 'number', required: true },
      { key: 'dosen_wali_id', label: 'Dosen wali', type: 'select', options: dosenOpts },
    ],
  },
  dosen: {
    title: 'Dosen',
    description: 'Dosen tetap dan pengampu. NIDN dipakai untuk login.',
    endpoint: '/dosen',
    singular: 'Dosen',
    columns: [
      { key: 'nidn', label: 'NIDN', mono: true, width: '130px' },
      { key: 'nama', label: 'Nama' },
      { key: 'email', label: 'Email', hideMobile: true },
      { key: 'prodi_nama', label: 'Homebase', hideMobile: true },
    ],
    filters: [{ key: 'prodi_id', label: 'Semua prodi', options: prodiOpts }],
    fields: [
      { key: 'nama', label: 'Nama lengkap', required: true, wide: true },
      { key: 'nidn', label: 'NIDN', required: true },
      { key: 'email', label: 'Email', type: 'email', required: true },
      { key: 'prodi_id', label: 'Homebase prodi', type: 'select', options: prodiOpts, required: true, wide: true },
    ],
  },
  prodi: {
    title: 'Program studi',
    description: 'Prodi menentukan jenjang dan batas masa studi (EC-09).',
    endpoint: '/prodi',
    singular: 'Prodi',
    columns: [
      { key: 'kode', label: 'Kode', mono: true, width: '100px' },
      { key: 'nama', label: 'Nama' },
      { key: 'jenjang', label: 'Jenjang' },
      { key: 'fakultas_nama', label: 'Fakultas', hideMobile: true },
      { key: 'masa_studi_maks', label: 'Masa studi maks', align: 'right', hideMobile: true, format: (v) => `${v} semester` },
    ],
    filters: [{ key: 'fakultas_id', label: 'Semua fakultas', options: fakultasOpts }],
    fields: [
      { key: 'kode', label: 'Kode', required: true },
      { key: 'jenjang', label: 'Jenjang', type: 'select', required: true, options: [{ value: 'D3', label: 'D3' }, { value: 'S1', label: 'S1' }, { value: 'S2', label: 'S2' }] },
      { key: 'nama', label: 'Nama', required: true, wide: true },
      { key: 'fakultas_id', label: 'Fakultas', type: 'select', options: fakultasOpts, required: true },
      { key: 'masa_studi_maks', label: 'Masa studi maks (semester)', type: 'number', required: true },
    ],
    defaults: { jenjang: 'S1', masa_studi_maks: 14 },
  },
  fakultas: {
    title: 'Fakultas',
    description: 'Fakultas menaungi prodi.',
    endpoint: '/fakultas',
    singular: 'Fakultas',
    columns: [
      { key: 'kode', label: 'Kode', mono: true, width: '120px' },
      { key: 'nama', label: 'Nama' },
    ],
    fields: [
      { key: 'kode', label: 'Kode', required: true },
      { key: 'nama', label: 'Nama', required: true },
    ],
  },
  ruang: {
    title: 'Ruang',
    description: 'Ruang kelas dipakai untuk deteksi bentrok jadwal (FR-15).',
    endpoint: '/ruang',
    singular: 'Ruang',
    columns: [
      { key: 'kode', label: 'Kode', mono: true, width: '120px' },
      { key: 'nama', label: 'Nama' },
      { key: 'kapasitas', label: 'Kapasitas', align: 'right' },
    ],
    fields: [
      { key: 'kode', label: 'Kode', required: true },
      { key: 'kapasitas', label: 'Kapasitas', type: 'number', required: true },
      { key: 'nama', label: 'Nama', required: true, wide: true },
    ],
  },
  periode: {
    title: 'Periode akademik',
    description: 'Hanya satu periode aktif per kampus. Tanggal tahapan dievaluasi di zona waktu kampus.',
    endpoint: '/periode',
    singular: 'Periode',
    canDelete: false,
    columns: [
      { key: 'kode', label: 'Kode', mono: true, width: '100px' },
      { key: 'tipe', label: 'Tipe', format: (v, r) => `${v} ${r.tahun}/${(r.tahun as number) + 1}` },
      { key: 'tahapan.krs_mulai', label: 'KRS', hideMobile: true, format: (_, r) => `${date((r.tahapan as Record<string, string>).krs_mulai)} – ${date((r.tahapan as Record<string, string>).krs_selesai)}` },
      { key: 'tahapan.nilai_selesai', label: 'Batas nilai', hideMobile: true, format: (v) => date(v as string) },
      { key: 'status', label: 'Status', status: 'periode' },
    ],
    rowActions: [
      {
        label: 'Jadikan aktif',
        icon: CalendarCheck2,
        show: (r) => r.status !== 'aktif',
        confirm: 'Aktifkan periode ini? Periode aktif sebelumnya akan diselesaikan.',
        run: (r) => api.post(`/periode/${r.id}/activate`),
        success: 'Periode diaktifkan',
      },
    ],
    fields: [
      { key: 'tahun', label: 'Tahun', type: 'number', required: true },
      { key: 'tipe', label: 'Tipe', type: 'select', required: true, options: [{ value: 'Ganjil', label: 'Ganjil' }, { value: 'Genap', label: 'Genap' }, { value: 'Pendek', label: 'Pendek' }] },
      { key: 'tahapan.krs_mulai', label: 'KRS mulai', type: 'date', required: true },
      { key: 'tahapan.krs_selesai', label: 'KRS selesai', type: 'date', required: true },
      { key: 'tahapan.revisi_mulai', label: 'Revisi mulai', type: 'date' },
      { key: 'tahapan.revisi_selesai', label: 'Revisi selesai', type: 'date' },
      { key: 'tahapan.kuliah_mulai', label: 'Kuliah mulai', type: 'date', required: true },
      { key: 'tahapan.kuliah_selesai', label: 'Kuliah selesai', type: 'date', required: true },
      { key: 'tahapan.uts', label: 'UTS', type: 'date', required: true },
      { key: 'tahapan.uas', label: 'UAS', type: 'date', required: true },
      { key: 'tahapan.nilai_selesai', label: 'Batas input nilai', type: 'date', required: true, wide: true },
    ],
    defaults: { tahun: thisYear, tipe: 'Ganjil', tahapan: {} },
  },
} satisfies Record<string, ResourceConfig>

export type ResourceKey = keyof typeof RESOURCES
