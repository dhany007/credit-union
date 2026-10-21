"use client";

import { createContext, useContext, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Moon, Sun } from "lucide-react";
import {
  translate,
  type Locale,
  type MessageKey,
  type Values,
} from "@/lib/messages";

export type Theme = "light" | "dark";
const localeKey = "credit-union.locale";
const themeKey = "credit-union.theme";

type Preferences = {
  locale: Locale;
  theme: Theme;
  ready: boolean;
  setLocale: (locale: Locale) => void;
  setTheme: (theme: Theme) => void;
};

const PreferencesContext = createContext<Preferences | null>(null);

function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
  document.documentElement.style.colorScheme = theme;
}

function save(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    /* Preferences still work when storage is blocked. */
  }
}

export function PreferencesProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [locale, updateLocale] = useState<Locale>("id");
  const [theme, updateTheme] = useState<Theme>("light");
  const [ready, setReady] = useState(false);

  useEffect(() => {
    let savedLocale: string | null = null;
    let savedTheme: string | null = null;
    try {
      savedLocale = localStorage.getItem(localeKey);
      savedTheme = localStorage.getItem(themeKey);
    } catch {
      /* Use defaults when browser storage is unavailable. */
    }
    const initialLocale = savedLocale === "en" ? "en" : "id";
    const initialTheme =
      savedTheme === "light" || savedTheme === "dark"
        ? savedTheme
        : window.matchMedia("(prefers-color-scheme: dark)").matches
          ? "dark"
          : "light";
    updateLocale(initialLocale);
    updateTheme(initialTheme);
    document.documentElement.lang = initialLocale;
    applyTheme(initialTheme);
    setReady(true);

    const onStorage = (event: StorageEvent) => {
      if (event.key === localeKey || event.key === null) {
        const next =
          event.key !== null && event.newValue === "en" ? "en" : "id";
        updateLocale(next);
        document.documentElement.lang = next;
      }
      if (event.key === themeKey || event.key === null) {
        const next =
          event.newValue === "dark" ||
          (event.newValue !== "light" &&
            window.matchMedia("(prefers-color-scheme: dark)").matches)
            ? "dark"
            : "light";
        updateTheme(next);
        applyTheme(next);
      }
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  function setLocale(next: Locale) {
    updateLocale(next);
    document.documentElement.lang = next;
    save(localeKey, next);
  }
  function setTheme(next: Theme) {
    updateTheme(next);
    applyTheme(next);
    save(themeKey, next);
  }

  return (
    <PreferencesContext.Provider
      value={{ locale, theme, ready, setLocale, setTheme }}
    >
      {children}
    </PreferencesContext.Provider>
  );
}

export function usePreferences() {
  const preferences = useContext(PreferencesContext);
  if (!preferences) throw new Error("PreferencesProvider is required");
  const intlLocale = preferences.locale === "en" ? "en-US" : "id-ID";
  return {
    ...preferences,
    t: (key: MessageKey, values?: Values) =>
      translate(preferences.locale, key, values),
    money: (amount: number) =>
      new Intl.NumberFormat(intlLocale, {
        style: "currency",
        currency: "IDR",
        maximumFractionDigits: 0,
      }).format(amount),
    date: (value: string) =>
      new Date(value).toLocaleString(intlLocale, {
        dateStyle: "medium",
        timeStyle: "short",
      }),
  };
}

export function PreferenceControls() {
  const { locale, theme, setLocale, setTheme, t } = usePreferences();
  return (
    <div className="no-print flex flex-wrap items-center gap-2">
      <label className="flex items-center gap-2 text-xs text-muted-foreground">
        <span className="sr-only">{t("Bahasa")}</span>
        <select
          aria-label={t("Bahasa")}
          value={locale}
          onChange={(event) =>
            setLocale(event.target.value === "en" ? "en" : "id")
          }
          className="h-9 rounded-md border bg-card px-2 text-xs text-foreground"
        >
          <option value="id">Bahasa Indonesia</option>
          <option value="en">English</option>
        </select>
      </label>
      <Button
        type="button"
        variant="outline"
        size="sm"
        aria-label={`${t("Tema")}: ${theme === "dark" ? t("Gelap") : t("Terang")}`}
        aria-pressed={theme === "dark"}
        onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
      >
        {theme === "dark" ? (
          <Moon aria-hidden="true" size={15} />
        ) : (
          <Sun aria-hidden="true" size={15} />
        )}
        {theme === "dark" ? t("Gelap") : t("Terang")}
      </Button>
    </div>
  );
}
