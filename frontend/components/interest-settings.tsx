"use client";

import { useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { localizedError } from "@/lib/api-errors";
import { nextJakartaDate, rateToBps } from "@/lib/interest";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";

type Band = { min_balance: number; max_balance: number | null; rate_bps: number };
type Policy = { id: number; effective_date: string; base_rate_bps: number; bands: Band[]; reason: string };
type Status = { enabled: boolean; start_date: string | null; last_completed_date: string | null; outstanding_months: string[]; failure_class: string };
type DraftBand = { minimum: string; maximum: string; rate: string };
const initialBands: DraftBand[] = [
  { minimum: "0", maximum: "1000000", rate: "3" },
  { minimum: "1000000", maximum: "10000000", rate: "4" },
  { minimum: "10000000", maximum: "100000000", rate: "4.5" },
  { minimum: "100000000", maximum: "", rate: "5" },
];

export function InterestSettings() {
  const { t, locale, money } = usePreferences();
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [status, setStatus] = useState<Status | null>(null);
  const [effective, setEffective] = useState(nextJakartaDate);
  const [baseRate, setBaseRate] = useState("3");
  const [bands, setBands] = useState<DraftBand[]>(initialBands);
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState(false);
  const pending = useRef<{ payload: string; key: string } | null>(null);
  async function load(signal?: AbortSignal) {
    const [versions, progress] = await Promise.all([
      api<{ items: Policy[] }>("/interest/policies", { signal }),
      api<Status>("/interest/status", { signal }),
    ]);
    if (signal?.aborted) return;
    setPolicies(versions.items);
    setStatus(progress);
  }
  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
 api<{ items: Policy[] }>("/interest/policies", { signal: controller.signal }),
 api<Status>("/interest/status", { signal: controller.signal }),
 ]).then(([versions, progress]) => {
 if (!controller.signal.aborted) { setPolicies(versions.items); setStatus(progress); }
 }).catch((e) => {
      if (!controller.signal.aborted) setError(e instanceof Error ? e.message : "");
    }).finally(() => { if (!controller.signal.aborted) setLoading(false); });
    return () => controller.abort();
  }, []);
  async function save(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(""); setSaved(false);
    const base = rateToBps(baseRate);
    const parsed = bands.map((band) => ({
      min_balance: Number(band.minimum),
      max_balance: band.maximum === "" ? null : Number(band.maximum),
      rate_bps: rateToBps(band.rate),
    }));
    if (base === null || parsed.some((band, i) =>
      bands[i].minimum.trim() === "" || !Number.isSafeInteger(band.min_balance) || band.min_balance < 0 ||
      (band.max_balance !== null && (!Number.isSafeInteger(band.max_balance) || band.max_balance <= band.min_balance)) ||
      band.rate_bps === null)) {
      setError("Invalid savings interest policy"); return;
    }
    const payload = JSON.stringify({
      effective_date: effective, base_rate_bps: base, bands: parsed,
      reason: reason.trim(), expected_latest_id: policies[0]?.id ?? null,
    });
    if (pending.current?.payload !== payload) pending.current = { payload, key: crypto.randomUUID() };
    setBusy(true);
    try {
      const policy = await api<Policy>("/interest/policies", {
        method: "POST", headers: { "Idempotency-Key": pending.current.key }, body: payload,
      });
      pending.current = null;
      setPolicies((current) => [policy, ...current.filter((item) => item.id !== policy.id)]);
      setSaved(true); setReason("");
      await load();
    } catch (e) { setError(e instanceof Error ? e.message : "Operation failed. Please retry with the same reference."); }
    finally { setBusy(false); }
  }
  function updateBand(index: number, field: keyof DraftBand, value: string) {
    setBands((current) => current.map((band, i) => i === index ? { ...band, [field]: value } : band));
    setSaved(false);
  }
  return <div className="grid gap-6">
    <Card><CardContent className="grid gap-3 pt-6 text-sm" aria-busy={loading}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="font-semibold">{t("Status bunga simpanan")}</h2>
        <Button variant="outline" disabled={busy || loading} onClick={async () => {
          setLoading(true); setError("");
          try { await load(); } catch (e) { setError(e instanceof Error ? e.message : ""); }
          finally { setLoading(false); }
        }}>{t("Muat ulang")}</Button>
      </div>
      {loading ? <p role="status">{t("Memuat...")}</p> : status && <>
        <p>{t(status.enabled ? "Perhitungan otomatis aktif" : "Perhitungan otomatis nonaktif")}</p>
        <dl className="grid gap-2 sm:grid-cols-3">
          <div><dt className="text-muted-foreground">{t("Tanggal aktivasi")}</dt><dd>{status.start_date ?? "—"}</dd></div>
          <div><dt className="text-muted-foreground">{t("Perhitungan terakhir")}</dt><dd>{status.last_completed_date ?? "—"}</dd></div>
          <div><dt className="text-muted-foreground">{t("Bulan belum dikreditkan")}</dt><dd>{status.outstanding_months.join(", ") || "—"}</dd></div>
        </dl>
        {status.failure_class && <p role="alert" className="text-destructive">{t("Perhitungan bunga gagal. Periksa log operasional.")}</p>}
      </>}
      <p className="text-muted-foreground">{t("Bunga dihitung dari saldo penutupan harian, dibagi 365, dan dibulatkan sekali saat kredit bulanan. Bunga tidak menambah kas KCP.")}</p>
    </CardContent></Card>
    {error && <p role="alert" className="text-sm text-destructive">{localizedError(locale, error)}</p>}
    {saved && <p role="status" className="text-sm">{t("Versi tarif berhasil disimpan.")}</p>}
    <Card><CardContent className="pt-6">
      <form onSubmit={save} className="grid gap-5">
        <h2 className="font-semibold">{t("Versi tarif baru")}</h2>
        <p className="text-sm text-muted-foreground">{t("Tarif berlaku paling cepat besok. Versi baru menggantikan jadwal pada tanggal yang sama tanpa mengubah riwayat.")}</p>
        <fieldset disabled={busy || loading || status === null} className="grid gap-5">
          <legend className="sr-only">{t("Pengaturan tarif")}</legend>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="grid gap-2 text-sm">{t("Tanggal berlaku")}<Input required type="date" min={nextJakartaDate()} value={effective} onChange={(e) => setEffective(e.target.value)} /></label>
            <label className="grid gap-2 text-sm">{t("Tarif global (% p.a.)")}<Input required inputMode="decimal" value={baseRate} onChange={(e) => setBaseRate(e.target.value)} /></label>
          </div>
          <div className="overflow-x-auto"><table className="w-full min-w-[540px] text-left text-sm">
            <caption className="mb-3 text-left text-muted-foreground">{t("Rentang memakai batas bawah termasuk dan batas atas tidak termasuk. Di luar rentang menggunakan tarif global.")}</caption>
            <thead><tr><th scope="col" className="p-2">{t("Saldo mulai (Rp)")}</th><th scope="col" className="p-2">{t("Saldo hingga (Rp)")}</th><th scope="col" className="p-2">{t("Bunga (% p.a.)")}</th><th scope="col" className="p-2">{t("Tindakan")}</th></tr></thead>
            <tbody>{bands.map((band, i) => <tr key={i}>
              <td className="p-2"><Input aria-label={t("Saldo mulai baris {row}", { row: i + 1 })} required type="number" min="0" step="1" value={band.minimum} onChange={(e) => updateBand(i, "minimum", e.target.value)} /></td>
              <td className="p-2"><Input aria-label={t("Saldo hingga baris {row}", { row: i + 1 })} type="number" min="1" step="1" placeholder={t("Tanpa batas")} value={band.maximum} onChange={(e) => updateBand(i, "maximum", e.target.value)} /></td>
              <td className="p-2"><Input aria-label={t("Bunga baris {row}", { row: i + 1 })} required inputMode="decimal" value={band.rate} onChange={(e) => updateBand(i, "rate", e.target.value)} /></td>
              <td className="p-2"><Button type="button" variant="ghost" aria-label={t("Hapus rentang {row}", { row: i + 1 })} onClick={() => setBands((current) => current.filter((_, index) => index !== i))}>{t("Hapus")}</Button></td>
            </tr>)}</tbody>
          </table></div>
          <Button type="button" variant="outline" className="justify-self-start" disabled={bands.length >= 20} onClick={() => setBands((current) => [...current, { minimum: "", maximum: "", rate: "3" }])}>{t("Tambah rentang")}</Button>
          <label className="grid gap-2 text-sm">{t("Alasan perubahan")}<Textarea required maxLength={500} value={reason} onChange={(e) => setReason(e.target.value)} /></label>
          <Button type="submit" className="justify-self-start">{t(busy ? "Menyimpan..." : "Simpan versi tarif")}</Button>
        </fieldset>
      </form>
    </CardContent></Card>
    <Card><CardContent className="grid gap-4 pt-6">
      <h2 className="font-semibold">{t("Riwayat tarif")}</h2>
      {!loading && policies.length === 0 && <p className="text-sm text-muted-foreground">{t("Belum ada tarif. Simpan versi pertama untuk menetapkan tanggal aktivasi.")}</p>}
      {policies.map((policy) => <article key={policy.id} className="grid gap-2 border-t pt-4 text-sm">
        <h3 className="font-medium">#{policy.id} · {policy.effective_date} · {t("Tarif global (% p.a.)")}: {(policy.base_rate_bps / 100).toFixed(2)}%</h3>
        <p className="text-muted-foreground">{policy.reason}</p>
        <ul className="grid gap-1">{policy.bands.map((band, i) => <li key={i}>{money(band.min_balance)} ≤ B {band.max_balance === null ? "" : `< ${money(band.max_balance)}`} · {(band.rate_bps / 100).toFixed(2)}% p.a.</li>)}</ul>
      </article>)}
    </CardContent></Card>
  </div>;
}
