"use client";
import { movementLabel, movementSign } from "@/lib/interest";

import { useState } from "react";
import { Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { usePreferences } from "@/components/preferences";
import { ServerRows } from "@/components/data-table";

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
type Query = Filters;
const initialFilters: Filters = { accountNumber: "", period: "today", startDate: "", endDate: "" };
const initialQuery = (): Query => ({ ...initialFilters });

function clearDateValidity(input: HTMLInputElement) {
  const endInput = input.form?.elements.namedItem("end_date");
  if (endInput instanceof HTMLInputElement) endInput.setCustomValidity("");
}

export function TransactionList() {
  const { t, money, locale } = usePreferences();
  const [draft, setDraft] = useState<Filters>(initialFilters);
  const [query, setQuery] = useState<Query>(initialQuery);
  const params=new URLSearchParams({period:query.period});
  if(query.accountNumber)params.set("account_number",query.accountNumber);
  if(query.period==="custom"){params.set("start_date",query.startDate);params.set("end_date",query.endDate);}
  const path="/transactions/search?"+params.toString();

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
            setQuery({ ...draft, startDate, endDate, accountNumber: draft.accountNumber.trim() });
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
          <Button type="submit">
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
          <p>{query.period==="custom" ? query.startDate+" — "+query.endDate+" · Asia/Jakarta (WIB)" : t("Zona waktu: Asia/Jakarta (WIB)")}</p>
        </div>
      </div>
      <ServerRows<TransactionRow> path={path} label={t("Riwayat transaksi simpanan")}>{(rows,dates)=>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[1080px] text-left text-sm">
              <caption className="py-2 text-left text-xs text-muted-foreground">{t("Riwayat transaksi simpanan")}{dates ? " · "+dates.start+" — "+dates.end+" · Asia/Jakarta (WIB)" : ""}</caption>
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
                {rows.map((row) => (
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
                        {t(movementLabel(row.kind))}
                      </Badge>
                    </td>
                    <td className="whitespace-nowrap p-4 text-right font-mono text-xs">
                      {movementSign(row.kind)}
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
                      <p className="mt-1 text-muted-foreground">{row.actor === "System" ? t("Sistem") : row.actor}</p>
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
                {rows.length === 0 && (
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
      }</ServerRows>
    </Card>
  );
}
