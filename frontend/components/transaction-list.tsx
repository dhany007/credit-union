"use client";

import { useEffect, useState } from "react";
import { Loader2, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { usePreferences } from "@/components/preferences";
import { api } from "@/lib/api";
import { localizedError } from "@/lib/api-errors";

type TransactionRow = {
  reference: string;
  member_id: number;
  member_name: string;
  member_number: string;
  account_number: string;
  kind: "deposit" | "withdrawal";
  amount: number;
  balance_before: number | null;
  balance_after: number | null;
  note: string;
  created_at: string;
  branch: string;
  actor: string;
};

type Period = "today" | "last_7_days" | "last_30_days" | "custom";
type Filters = { accountNumber: string; period: Period; startDate: string; endDate: string };
type Query = Filters & { cursor: string; previous: string[] };
type Page = { items: TransactionRow[]; next_cursor: string; start_date: string; end_date: string; time_zone: string };
const initialFilters: Filters = { accountNumber: "", period: "today", startDate: "", endDate: "" };
const initialQuery = (): Query => ({ ...initialFilters, cursor: "", previous: [] });
type Result = { query: Query | null; rows: TransactionRow[]; page: Page | null; error: string };

function clearDateValidity(input: HTMLInputElement) {
  const endInput = input.form?.elements.namedItem("end_date");
  if (endInput instanceof HTMLInputElement) endInput.setCustomValidity("");
}

export function TransactionList() {
  const { t, money, locale } = usePreferences();
  const [draft, setDraft] = useState<Filters>(initialFilters);
  const [query, setQuery] = useState<Query>(initialQuery);
  const [result, setResult] = useState<Result>({
    query: null,
    page: null,
    rows: [],
    error: "",
  });
  const loading = result.query !== query;

  useEffect(() => {
    const controller = new AbortController();
    const params = new URLSearchParams({ period: query.period });
    if (query.accountNumber) params.set("account_number", query.accountNumber);
    if (query.period === "custom") {
      params.set("start_date", query.startDate);
      params.set("end_date", query.endDate);
    }
    if (query.cursor) params.set("cursor", query.cursor);
    api<Page>(`/transactions/search?${params}`, { signal: controller.signal })
      .then((page) => {
        if (!controller.signal.aborted) setResult({ query, rows: page.items, page, error: "" });
      })
      .catch((error) => {
        if (!controller.signal.aborted)
          setResult({
            query,
            rows: [],
            page: null,
            error: error instanceof Error ? error.message : "Unexpected error",
          });
      });
    return () => controller.abort();
  }, [query]);

  function reset() {
    setDraft(initialFilters);
    setQuery(initialQuery());
  }

  return (
    <Card className="gap-0 overflow-hidden py-0">
      <div className="grid gap-5 border-b p-5">
        <p className="text-xs text-muted-foreground">{t("Saldo sebelum dan sesudah adalah saldo simpanan anggota. Tanda — berarti snapshot historis tidak tersedia.")}</p>
        <form
          className="flex flex-wrap items-end gap-3"
          onSubmit={(event) => {
            event.preventDefault();
                        const form = new FormData(event.currentTarget);
            const startDate = String(form.get("start_date") ?? "");
            const endDate = String(form.get("end_date") ?? "");
            const endInput = event.currentTarget.elements.namedItem("end_date");
            if (draft.period === "custom" && startDate > endDate && endInput instanceof HTMLInputElement) {
              endInput.setCustomValidity(t("Tanggal akhir harus sama atau setelah tanggal mulai."));
              endInput.reportValidity();
              return;
            }
            setQuery({ ...draft, startDate, endDate, accountNumber: draft.accountNumber.trim(), cursor: "", previous: [] });
          }}
        >
          <label className="grid w-full gap-2 text-sm sm:w-72">
            <span className="font-medium">{t("Nomor rekening")}</span>
            <Input
              value={draft.accountNumber}
              onChange={(event) => setDraft({ ...draft, accountNumber: event.target.value })}
              inputMode="numeric"
              pattern="[1-9][0-9]{9}"
              minLength={10}
              maxLength={10}
              placeholder={t("Kosongkan untuk semua rekening")}
            />
          </label>
          <label className="grid gap-2 text-sm">
            <span className="font-medium">{t("Periode")}</span>
            <select
              className="h-9 rounded-md border bg-card px-3 text-sm text-foreground"
              value={draft.period}
              onChange={(event) => setDraft({ ...draft, period: event.target.value as Period })}
            >
              <option value="today">{t("Hari ini")}</option>
              <option value="last_7_days">{t("7 hari terakhir")}</option>
              <option value="last_30_days">{t("30 hari terakhir")}</option>
              <option value="custom">{t("Rentang khusus")}</option>
            </select>
          </label>
          {draft.period === "custom" && <>
            <label className="grid gap-2 text-sm">
              <span className="font-medium">{t("Tanggal mulai")}</span>
              <Input type="date" name="start_date" required defaultValue={draft.startDate} min="0001-01-01" max="9999-12-31"
                onInput={(event) => clearDateValidity(event.currentTarget)} />
            </label>
            <label className="grid gap-2 text-sm">
              <span className="font-medium">{t("Tanggal akhir")}</span>
              <Input type="date" name="end_date" required defaultValue={draft.endDate} min="0001-01-01" max="9999-12-31"
                onInput={(event) => clearDateValidity(event.currentTarget)} />
            </label>
          </>}
          <Button type="submit" disabled={loading}>
            <Search size={15} />
            {t("Terapkan filter")}
          </Button>
          <Button type="button" variant="outline" onClick={reset}>
            {t("Reset")}
          </Button>
        </form>
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-muted-foreground">
          <p>
            {query.accountNumber
              ? t("Rekening: {account}", { account: query.accountNumber })
              : t("Semua rekening · seluruh KCP")}
          </p>
          <p>{!loading && result.page ? `${result.page.start_date} — ${result.page.end_date} · Asia/Jakarta (WIB)` : t("Zona waktu: Asia/Jakarta (WIB)")}</p>
        </div>
      </div>
      {loading ? (
        <div
          role="status"
          className="flex items-center justify-center gap-2 p-12 text-sm text-muted-foreground"
        >
          <Loader2 className="animate-spin" aria-hidden="true" size={18} />
          {t("Memuat transaksi...")}
        </div>
      ) : result.error ? (
        <div
          role="alert"
          className="grid justify-items-center gap-4 p-8 text-sm"
        >
          <p className="text-destructive">
            {localizedError(locale, result.error)}
          </p>
          <Button
            type="button"
            variant="outline"
            onClick={() => setQuery({ ...query })}
          >
            {t("Coba lagi")}
          </Button>
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[1080px] text-left text-sm">
              <caption className="sr-only">
                {t("Riwayat transaksi simpanan")}
              </caption>
              <thead className="bg-secondary/40 text-[11px] uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th scope="col" className="p-4 pl-5 font-medium">
                    {t("Tanggal")}
                  </th>
                  <th scope="col" className="p-4 font-medium">
                    {t("Anggota / rekening")}
                  </th>
                  <th scope="col" className="p-4 font-medium">
                    {t("Jenis transaksi")}
                  </th>
                  <th scope="col" className="p-4 text-right font-medium">
                    {t("Nominal (Rp)")}
                  </th>
                  <th scope="col" className="p-4 text-right font-medium">{t("Saldo sebelum")}</th>
                  <th scope="col" className="p-4 text-right font-medium">{t("Saldo sesudah")}</th>
                  <th scope="col" className="p-4 font-medium">
                    {t("KCP / petugas")}
                  </th>
                  <th scope="col" className="p-4 pr-5 font-medium">
                    {t("Referensi / catatan")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {result.rows.map((row) => (
                  <tr key={row.reference} className="border-t align-top">
                    <td className="whitespace-nowrap p-4 pl-5 text-xs">
                      {new Intl.DateTimeFormat(locale === "en" ? "en-US" : "id-ID", { dateStyle: "medium", timeStyle: "short", timeZone: "Asia/Jakarta" }).format(new Date(row.created_at))}
                    </td>
                    <td className="p-4">
                      <p className="font-medium">{row.member_name}</p>
                      <p className="mt-1 font-mono text-xs">
                        {row.account_number}
                      </p>
                      <p className="mt-1 text-[11px] text-muted-foreground">
                        {row.member_number}
                      </p>
                    </td>
                    <td className="p-4">
                      <Badge variant="outline">
                        {row.kind === "deposit"
                          ? t("Setoran tunai")
                          : t("Penarikan tunai")}
                      </Badge>
                    </td>
                    <td className="whitespace-nowrap p-4 text-right font-mono text-xs">
                      {row.kind === "deposit" ? "+" : "−"}
                      {money(row.amount)}
                    </td>
                    <td className="whitespace-nowrap p-4 text-right font-mono text-xs" title={row.balance_before == null ? t("Saldo historis tidak tersedia") : undefined}>
                      {row.balance_before == null ? "—" : money(row.balance_before)}
                    </td>
                    <td className="whitespace-nowrap p-4 text-right font-mono text-xs" title={row.balance_after == null ? t("Saldo historis tidak tersedia") : undefined}>
                      {row.balance_after == null ? "—" : money(row.balance_after)}
                    </td>
                    <td className="p-4 text-xs">
                      <p>{row.branch}</p>
                      <p className="mt-1 text-muted-foreground">{row.actor}</p>
                    </td>
                    <td className="p-4 pr-5">
                      <p className="break-all font-mono text-[10px]">
                        {row.reference}
                      </p>
                      {row.note && (
                        <p className="mt-2 max-w-xs break-words text-xs text-muted-foreground">
                          {row.note}
                        </p>
                      )}
                    </td>
                  </tr>
                ))}
                {result.rows.length === 0 && (
                  <tr>
                    <td
                      colSpan={8}
                      className="p-12 text-center text-muted-foreground"
                    >
                      {query.accountNumber
                        ? t("Tidak ada transaksi pada periode dan rekening ini.")
                        : t("Tidak ada transaksi pada periode ini.")}
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
          <div className="flex flex-wrap items-center justify-between gap-3 border-t px-5 py-4 text-xs text-muted-foreground">
            <p role="status">{t("Halaman {page} · {count} transaksi", { page: query.previous.length + 1, count: result.rows.length })}</p>
            <div className="flex gap-2">
              <Button type="button" variant="outline" size="sm" disabled={query.previous.length === 0}
                onClick={() => {
                  if (!result.page) return;
                  setQuery({ ...query, period: "custom", startDate: result.page.start_date, endDate: result.page.end_date,
                    cursor: query.previous[query.previous.length - 1], previous: query.previous.slice(0, -1) });
                }}>{t("Sebelumnya")}</Button>
              <Button type="button" variant="outline" size="sm" disabled={!result.page?.next_cursor}
                onClick={() => {
                  if (!result.page?.next_cursor) return;
                  setQuery({ ...query, period: "custom", startDate: result.page.start_date, endDate: result.page.end_date,
                    cursor: result.page.next_cursor, previous: [...query.previous, query.cursor] });
                }}>{t("Berikutnya")}</Button>
            </div>
          </div>        </>
      )}
    </Card>
  );
}
