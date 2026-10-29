package seed

var firstNames = []string{"Adi", "Ayu", "Bagus", "Bima", "Citra", "Dewi", "Dimas", "Eka", "Fajar", "Fitri", "Gilang", "Hana", "Indah", "Joko", "Kartika",
	"Lestari", "Made", "Nadia", "Nanda", "Oki", "Putri", "Putu", "Raka", "Rina", "Rizky", "Sari", "Satria", "Tari", "Teguh", "Udin", "Vina", "Wahyu",
	"Wulan", "Yoga", "Yuni", "Zaki", "Arif", "Bayu", "Dian", "Gita", "Hendra", "Intan", "Kevin", "Laras", "Maya", "Nur", "Reza", "Siti", "Taufik", "Andi"}

var lastNames = []string{"Pratama", "Saputra", "Wijaya", "Santoso", "Lestari", "Hidayat", "Kusuma", "Nugroho", "Rahmawati", "Setiawan", "Siregar", "Utami",
	"Wibowo", "Halim", "Gunawan", "Permana", "Ramadhan", "Purnomo", "Hakim", "Syahputra", "Anggraini", "Firmansyah", "Maharani", "Sitorus", "Tanjung",
	"Pangestu", "Kurniawan", "Susanto", "Rahman", "Fauzi"}

var gelar = []string{"Dr.", "", "", "Prof. Dr.", ""}

// course banks: 8 semesters × 5 courses (D3 uses the first 6 semesters)
var bankTI = [][]string{
	{"Algoritma dan Pemrograman", "Matematika Diskrit", "Pengantar Teknologi Informasi", "Bahasa Indonesia", "Pendidikan Pancasila"},
	{"Struktur Data", "Kalkulus", "Sistem Digital", "Bahasa Inggris", "Pendidikan Kewarganegaraan"},
	{"Pemrograman Berorientasi Objek", "Aljabar Linear", "Organisasi Komputer", "Statistika", "Etika Profesi"},
	{"Basis Data", "Sistem Operasi", "Jaringan Komputer", "Interaksi Manusia dan Komputer", "Metode Numerik"},
	{"Rekayasa Perangkat Lunak", "Kecerdasan Buatan", "Pemrograman Web", "Keamanan Informasi", "Teori Bahasa dan Otomata"},
	{"Pengujian Perangkat Lunak", "Pembelajaran Mesin", "Komputasi Awan", "Grafika Komputer", "Kewirausahaan"},
	{"Proyek Perangkat Lunak", "Sistem Terdistribusi", "Metodologi Penelitian", "Pemrograman Mobile", "Data Mining"},
	{"Skripsi", "Kerja Praktik", "Manajemen Proyek TI", "Etika Komputer Lanjut", "Seminar Informatika"},
}

var bankSI = [][]string{
	{"Pengantar Sistem Informasi", "Logika dan Algoritma", "Matematika Bisnis", "Bahasa Indonesia", "Pendidikan Pancasila"},
	{"Pemrograman Dasar", "Statistika Bisnis", "Konsep Manajemen", "Bahasa Inggris", "Pendidikan Kewarganegaraan"},
	{"Analisis Proses Bisnis", "Basis Data", "Akuntansi Dasar", "Struktur Data", "Etika Profesi"},
	{"Analisis dan Perancangan Sistem", "Sistem Basis Data Lanjut", "Pemrograman Web", "Jaringan Komputer", "Manajemen Rantai Pasok"},
	{"Enterprise Resource Planning", "Manajemen Proyek SI", "Audit Sistem Informasi", "E-Business", "Interaksi Manusia dan Komputer"},
	{"Tata Kelola TI", "Business Intelligence", "Keamanan Sistem Informasi", "Sistem Pendukung Keputusan", "Kewirausahaan"},
	{"Perencanaan Strategis SI", "Data Warehouse", "Metodologi Penelitian", "Manajemen Layanan TI", "Analitika Data"},
	{"Skripsi", "Kerja Praktik", "Seminar Sistem Informasi", "Inovasi Digital", "Etika Bisnis"},
}

var bankMN = [][]string{
	{"Pengantar Manajemen", "Pengantar Ekonomi Mikro", "Matematika Ekonomi", "Bahasa Indonesia", "Pendidikan Pancasila"},
	{"Manajemen Lanjutan", "Pengantar Ekonomi Makro", "Statistika Ekonomi", "Bahasa Inggris Bisnis", "Pendidikan Kewarganegaraan"},
	{"Manajemen Keuangan I", "Manajemen Pemasaran I", "Akuntansi Manajemen", "Hukum Bisnis", "Etika Bisnis"},
	{"Manajemen Keuangan II", "Manajemen Pemasaran II", "Manajemen Operasional", "Perilaku Organisasi", "Sistem Informasi Manajemen"},
	{"Manajemen Investasi", "Riset Pemasaran", "Manajemen SDM", "Ekonomi Manajerial", "Perpajakan"},
	{"Manajemen Risiko", "Pemasaran Digital", "Manajemen Strategik", "Studi Kelayakan Bisnis", "Kewirausahaan"},
	{"Seminar Keuangan", "Metodologi Penelitian", "Manajemen Perubahan", "Bisnis Internasional", "Kepemimpinan"},
	{"Skripsi", "Magang", "Seminar Manajemen", "Tata Kelola Perusahaan", "Etika Profesi"},
}

var bankAK = [][]string{
	{"Pengantar Akuntansi I", "Pengantar Bisnis", "Matematika Keuangan", "Bahasa Indonesia", "Pendidikan Pancasila"},
	{"Pengantar Akuntansi II", "Ekonomi Mikro", "Statistika", "Bahasa Inggris", "Pendidikan Kewarganegaraan"},
	{"Akuntansi Keuangan Menengah I", "Akuntansi Biaya", "Perpajakan I", "Aplikasi Komputer Akuntansi", "Hukum Komersial"},
	{"Akuntansi Keuangan Menengah II", "Akuntansi Manajemen", "Perpajakan II", "Sistem Informasi Akuntansi", "Etika Profesi"},
	{"Auditing", "Akuntansi Sektor Publik", "Penganggaran", "Praktikum Akuntansi", "Kewirausahaan"},
	{"Tugas Akhir", "Praktik Kerja Lapangan", "Analisis Laporan Keuangan", "Seminar Akuntansi", "Akuntansi Perbankan"},
}

var bankTM = [][]string{
	{"Gambar Teknik", "Fisika Terapan", "Matematika Teknik I", "Bahasa Indonesia", "Pendidikan Pancasila"},
	{"Mekanika Teknik", "Matematika Teknik II", "Material Teknik", "Bahasa Inggris Teknik", "Pendidikan Kewarganegaraan"},
	{"Elemen Mesin I", "Termodinamika", "Proses Produksi I", "Kelistrikan Dasar", "K3 Industri"},
	{"Elemen Mesin II", "Mekanika Fluida", "Proses Produksi II", "CAD/CAM", "Metrologi"},
	{"Perawatan Mesin", "Sistem Kontrol", "Motor Bakar", "Praktikum CNC", "Kewirausahaan"},
	{"Tugas Akhir", "Praktik Kerja Lapangan", "Manajemen Industri", "Seminar Teknik Mesin", "Energi Terbarukan"},
}

var bankTID3 = [][]string{
	{"Algoritma dan Pemrograman", "Matematika Diskrit", "Pengantar Teknologi Informasi", "Bahasa Indonesia", "Pendidikan Pancasila"},
	{"Struktur Data", "Basis Data", "Sistem Digital", "Bahasa Inggris", "Pendidikan Kewarganegaraan"},
	{"Pemrograman Web", "Jaringan Komputer", "Sistem Operasi", "Statistika", "Etika Profesi"},
	{"Pemrograman Mobile", "Rekayasa Perangkat Lunak", "Administrasi Server", "Desain Antarmuka", "Basis Data Lanjut"},
	{"Proyek Aplikasi", "Keamanan Jaringan", "Komputasi Awan", "Pengujian Perangkat Lunak", "Kewirausahaan"},
	{"Tugas Akhir", "Praktik Kerja Lapangan", "Seminar TI", "Manajemen Proyek", "DevOps Dasar"},
}
