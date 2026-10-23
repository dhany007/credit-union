import type { Locale } from "@/lib/messages";

const errors: Record<string, string> = {
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
