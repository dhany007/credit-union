"use client";

import { Toaster as Sonner } from "sonner";
import { usePreferences } from "@/components/preferences";

export function Toaster() {
  const { theme, t } = usePreferences();
  return <Sonner theme={theme} position="top-right" closeButton duration={5000}
    containerAriaLabel={t("Notifikasi")}
    toastOptions={{
      style: {
        background: "var(--popover)",
        color: "var(--popover-foreground)",
        borderColor: "var(--border)",
        borderRadius: "var(--radius)",
        fontFamily: "inherit",
      },
    }}/>;
}
