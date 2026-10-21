import type { Metadata } from "next";
import "./globals.css";
export const metadata: Metadata = { title: "Credit Union · Layanan KCP", description: "Layanan anggota, kas, dan pengajuan pinjaman koperasi." };
export default function RootLayout({ children }: Readonly<{children: React.ReactNode}>) { return <html lang="id"><body>{children}</body></html>; }
