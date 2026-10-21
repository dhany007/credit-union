import type { Metadata } from "next";
import { PreferencesProvider } from "@/components/preferences";
import "./globals.css";

// Apply the stored/system theme before paint; the root attribute mismatch is intentional.
const themeBootstrap = `(()=>{let theme;try{theme=localStorage.getItem('credit-union.theme')}catch{}const dark=theme==='dark'||(theme!=='light'&&matchMedia('(prefers-color-scheme: dark)').matches);document.documentElement.classList.toggle('dark',dark);document.documentElement.style.colorScheme=dark?'dark':'light'})()`;

export const metadata: Metadata = {
  title: "Credit Union",
  description: "Credit Union · Layanan KCP / Branch services",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="id" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeBootstrap }} />
      </head>
      <body>
        <PreferencesProvider>{children}</PreferencesProvider>
      </body>
    </html>
  );
}
