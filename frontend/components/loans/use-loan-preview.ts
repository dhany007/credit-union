"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { scheduleLoanPreview } from "@/lib/loan-preview";
import type { Preview } from "@/lib/loans";

export function useLoanPreview(path: string, input: Record<string, string | number> | null) {
  const [attempt, setAttempt] = useState(0);
  const [result, setResult] = useState<{ key: string; preview?: Preview; error?: string }>({ key: "" });
  const payload = input === null ? "" : JSON.stringify(input);
  const key = path + ":" + payload + ":" + attempt;

  useEffect(() => {
    if (!payload) return;
    return scheduleLoanPreview(
      signal => api<Preview>(path, { method: "POST", body: payload, signal }),
      preview => setResult({ key, preview }),
      error => setResult({ key, error: error instanceof Error ? error.message : "Failed to fetch" }),
    );
  }, [path, payload, key]);

  const preview = payload && result.key === key ? result.preview : undefined;
  const error = payload && result.key === key ? result.error ?? "" : "";
  return { preview, error, loading: !!payload && !preview && !error, retry: () => setAttempt(value => value + 1) };
}
