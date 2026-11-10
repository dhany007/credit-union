import type { Locale } from "@/lib/messages";

const errors: Record<string, string> = {
  "Request reference storage unavailable": "Penyimpanan referensi permintaan tidak tersedia. Aktifkan penyimpanan sesi sebelum bertransaksi.",
  "Invalid request reference": "Referensi permintaan tersimpan tidak valid. Hubungi pengelola sebelum bertransaksi.",
  "Invalid loan interest policy": "Aturan bunga pinjaman tidak valid. Bunga minimal 3% p.a. dan batas nominal harus valid.",
  "Configure an effective loan interest policy first": "Atur tarif bunga pinjaman yang sudah berlaku terlebih dahulu.",
  "Loan base rate must be at least 3% p.a.; provide a valid bands array and reason": "Tarif dasar minimal 3% p.a.; isi rentang dan alasan yang valid.",
  "Loan band rate must be at least 3% p.a. with valid amount bounds": "Tarif rentang minimal 3% p.a. dengan batas nominal yang valid.",
  "Loan interest bands overlap": "Rentang bunga pinjaman bertumpuk.",
  "Loan policy must take effect tomorrow or later": "Tarif pinjaman harus berlaku paling cepat besok.",
  "Loan policy changed; reload before saving": "Tarif pinjaman berubah; muat ulang sebelum menyimpan.",
  "Invalid policy effective date": "Tanggal berlaku tarif tidak valid.",
  "Invalid loan principal or tenor": "Pokok atau tenor pinjaman tidak valid.",
  "Invalid loan principal, annual rate or tenor": "Pokok, tarif tahunan, atau tenor tidak valid.",
  "Invalid expected disbursement date": "Perkiraan tanggal pencairan tidak valid.",
  "Expected payout date must be within the next year": "Perkiraan pencairan harus dalam satu tahun ke depan.",
  "Principal is too small for the selected tenor and rounding": "Pokok terlalu kecil untuk tenor dan pembulatan yang dipilih.",
  "Loan calculation exceeds the supported limit": "Perhitungan pinjaman melebihi batas yang didukung.",
  "Approved terms and reviewed policy are required": "Syarat yang disetujui dan preview tarif wajib tersedia.",
  "Approval revision changed": "Revisi persetujuan berubah; muat ulang dan tinjau kembali.",
  "Loan pricing changed; review a new preview": "Tarif pinjaman berubah; buat dan tinjau preview baru.",
  "Application has already been disbursed": "Pengajuan sudah dicairkan.",
  "Member is inactive": "Anggota tidak aktif.",
  "Responsible KCP is inactive": "KCP penanggung jawab tidak aktif.",
  "A complete priced approval is required before disbursement": "Persetujuan lengkap dengan tarif wajib tersedia sebelum pencairan.",
  "Current payout authority is not configured": "Kewenangan pencairan saat ini belum dikonfigurasi.",
  "Payout attestations and verification note are required": "Verifikasi penerima, perjanjian, persyaratan, dan catatan wajib diisi.",
  "Payout terms changed; review again": "Syarat pencairan berubah; tinjau kembali.",
  "Payout date changed; review the schedule again": "Tanggal pencairan berubah; tinjau kembali jadwal.",
  "Loan account not found": "Rekening pinjaman tidak ditemukan.",
  "Loan resource not found": "Data pinjaman tidak ditemukan.",
  "Schedule version not found": "Versi jadwal tidak ditemukan.",
  "Invalid repayment amount": "Nominal pembayaran tidak valid.",
  "Loan is not active": "Pinjaman tidak aktif.",
  "Repayment exceeds remaining scheduled payable": "Pembayaran melebihi sisa kewajiban terjadwal.",
  "Repayment note is too long": "Catatan pembayaran terlalu panjang.",
  "Payment terms changed; review allocation again": "Data pembayaran berubah; tinjau kembali alokasi.",
  "Teller access required": "Akses Teller diperlukan.",
  "Supervisor access required": "Akses Supervisor diperlukan.",
  "Invalid loan filter": "Filter pinjaman tidak valid.",
  "Invalid account number": "Nomor rekening tidak valid.",
  "Invalid loan status": "Status pinjaman tidak valid.",
  "Invalid payout filter": "Filter pencairan tidak valid.",
  "Invalid application reference filter": "Filter referensi pengajuan tidak valid.",
  "Invalid loan reference filter": "Filter referensi pinjaman tidak valid.",
  "Loan reference is too long": "Referensi pinjaman terlalu panjang.",
  "Invalid pagination cursor; apply the filters again": "Halaman tidak valid; terapkan filter kembali.",
  "Invalid loan resource ID": "ID pinjaman tidak valid.",
 "Savings business date is already closed": "Tanggal pembukuan simpanan sudah ditutup",
 "Interest policy must take effect tomorrow or later without changing activation": "Tarif harus berlaku paling cepat besok dan tidak boleh memajukan tanggal aktivasi",
 "Savings interest policy changed. Reload and try again": "Aturan bunga berubah. Muat ulang lalu coba lagi",
 "Savings interest bands overlap or are not ordered": "Rentang bunga simpanan bertumpuk atau belum berurutan",
 "Invalid savings interest bands": "Rentang bunga simpanan tidak valid",
 "Invalid savings interest policy": "Aturan bunga simpanan tidak valid",
  "Invalid date filter. Check period and start/end dates": "Filter tanggal tidak valid. Periksa periode dan tanggal mulai/akhir",
  "Invalid pagination cursor. Apply the filters again": "Halaman tidak valid. Terapkan kembali filter",
  "Invalid account number. Enter exactly 10 digits":
    "Nomor rekening tidak valid. Masukkan tepat 10 digit",
  "Savings account not found": "Rekening simpanan tidak ditemukan",
  "Your role cannot perform this action":
    "Peran Anda tidak berwenang melakukan tindakan ini",
  "Check NIK, name, phone, and address":
    "Periksa NIK, nama, nomor telepon, dan alamat",
  "Search is too long": "Pencarian terlalu panjang",
  "Invalid member": "Anggota tidak valid",
  "Invalid member, amount, or transaction type":
    "Anggota, nominal, atau jenis transaksi tidak valid",
  "Member not found": "Anggota tidak ditemukan",
  "Insufficient member balance": "Saldo anggota tidak mencukupi",
  "Insufficient cash at this KCP": "Kas pada KCP ini tidak mencukupi",
  "Balance exceeds the supported limit": "Saldo melebihi batas yang didukung",
  "Invalid application": "Pengajuan tidak valid",
  "Check loan amount, tenor, purpose, and risk":
    "Periksa nominal, tenor, tujuan, dan risiko pinjaman",
  "Invalid member or responsible KCP":
    "Anggota atau KCP penanggung jawab tidak valid",
  "A decision and reason of at least 5 characters are required":
    "Keputusan dan alasan minimal 5 karakter wajib diisi",
  "Application not found": "Pengajuan tidak ditemukan",
  "Application already decided": "Pengajuan sudah diputuskan",
  "You cannot decide your own application input":
    "Anda tidak dapat memutuskan pengajuan yang Anda input sendiri",
  "No approval authority matches this amount, product, and risk. Configure policy first":
    "Tidak ada kewenangan persetujuan untuk nominal, produk, dan risiko ini. Konfigurasikan aturan terlebih dahulu",
  "Please sign in": "Silakan masuk",
  "Too many sign-in attempts. Try again later":
    "Terlalu banyak percobaan masuk. Coba lagi nanti",
  "Incorrect username or password": "Username atau kata sandi salah",
  "Session expired. Please sign in": "Sesi berakhir. Silakan masuk kembali",
  "Authentication service unavailable": "Layanan autentikasi tidak tersedia",
  "A valid transaction reference is required":
    "Referensi transaksi yang valid wajib diisi",
  "Reference already used for different data":
    "Referensi sudah digunakan untuk data yang berbeda",
  "Invalid request data": "Data permintaan tidak valid",
  "Only one JSON object is allowed": "Hanya satu objek JSON yang diperbolehkan",
  "Invalid request origin":
    "Asal permintaan tidak valid. Gunakan alamat localhost yang dikonfigurasi",
  "Operation failed. Please retry with the same reference.":
    "Operasi gagal. Coba lagi dengan referensi yang sama.",
  "Data already exists. Check the NIK or request reference.":
    "Data sudah terdaftar. Periksa NIK atau referensi permintaan.",
  "The server could not be reached. Please try again.":
    "Server tidak dapat dihubungi. Coba lagi.",
  "Failed to fetch": "Server tidak dapat dihubungi. Coba lagi.",
  "Load failed": "Server tidak dapat dihubungi. Coba lagi.",
};

// Keep public API messages unchanged; unknown failures receive a safe localized fallback.
export function localizedError(locale: Locale, message: string) {
  if (Object.hasOwn(errors, message)) return locale === "id" ? errors[message] : message;
  return locale === "id"
    ? "Terjadi kesalahan. Silakan coba lagi."
    : "Something went wrong. Please try again.";
}
