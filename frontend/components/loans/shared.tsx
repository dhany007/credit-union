"use client";
import type { Allocation } from "@/lib/loans";
import { DetailList } from "@/components/ui/detail-list";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
export { ErrorNotice } from "@/components/modal";
import { usePreferences } from "@/components/preferences";
import { LocalRows, ServerRows } from "@/components/data-table";
import type { Installment, Preview } from "@/lib/loans";
export const selectClass = "h-10 rounded-md border bg-card px-3 text-sm w-full";
export function Field({ label, children }: {
    label: string;
    children: React.ReactNode;
}) {
    return <label className="grid gap-2 text-sm">
    <span className="font-medium">{label}</span>{children}</label>;
}
export function useRead<T>(path: string, attempt = 0) {
    const [state, setState] = useState<{
        path: string;
        data?: T;
        error?: string;
    }>({ path: "" });
    useEffect(() => {
        if (!path)
            return;
        const c = new AbortController();
        api<T>(path, { signal: c.signal }).then(data => {
            if (!c.signal.aborted)
                setState({ path, data });
        }).catch(e => {
            if (!c.signal.aborted)
                setState({ path, error: e instanceof Error ? e.message : "Failed to fetch" });
        });
        return () => c.abort();
    }, [path, attempt]);
    return state.path === path ? state : { path };
}
export function Paged<T>({ path, children, caption }: {
    path: string; children: (items:T[])=>React.ReactNode; caption:string;
}) { return <ServerRows<T> path={path} label={caption}>{children}</ServerRows>; }
export function ScheduleTable({ rows, actual = false }: {
    rows: Installment[];
    actual?: boolean;
}) {
    const { t, money } = usePreferences();
    const headings = [t("Angsuran"), t("Jatuh tempo"), ...(!actual ? [t("Pokok awal")] : []), t("Pokok"), t("Bunga"), t("Total"), ...(actual ? [t("Pokok terbayar"), t("Bunga terbayar"), t("Belum dibayar")] : [t("Pokok akhir")])];
    return <div className="overflow-x-auto">
    <table className="w-full whitespace-nowrap text-left text-sm">
    <caption className="py-3 text-left text-muted-foreground">{t(actual ? "Jadwal kontrak dan pembayaran aktual" : "Proyeksi jadwal · angsuran terakhir menyesuaikan pembulatan")}</caption>
    <thead>
    <tr>{headings.map(k => <th key={k} className="border-b p-3">{k}</th>)}</tr>
    </thead>
    <tbody>{rows.map(r => <tr key={r.installment_number}>
        <td className="border-b p-3">{r.installment_number}</td>
        <td className="border-b p-3">{r.due_date.slice(0, 10)}</td>{!actual && <td className="border-b p-3">{money(r.opening_principal ?? 0)}</td>}<td className="border-b p-3">{money(r.principal_due)}</td>
        <td className="border-b p-3">{money(r.interest_due)}</td>
        <td className="border-b p-3">{money(r.total_due)}</td>{actual ? <>
            <td className="border-b p-3">{money(r.principal_paid ?? 0)}</td>
            <td className="border-b p-3">{money(r.interest_paid ?? 0)}</td>
            <td className="border-b p-3">{money((r.principal_unpaid ?? 0) + (r.interest_unpaid ?? 0))}</td>
            </> : <td className="border-b p-3">{money(r.closing_principal ?? 0)}</td>}</tr>)}</tbody>
    </table>
    </div>;
}
export function SchedulePreview({ preview, showSummary = true, loading = false, stale = false, statusText }: {
    preview: Preview;
    showSummary?: boolean;
    loading?: boolean;
    stale?: boolean;
    statusText?: string;
}) {
    const { t, money } = usePreferences();
    return <section className="space-y-3 rounded-lg border p-4">
    <h3 className="font-semibold">{t("Estimasi jadwal angsuran")}</h3>
    <p className="text-sm text-muted-foreground">{t("Estimasi jadwal; tanggal aktual dan pembulatan final ditentukan saat pencairan.")}</p>
    {showSummary && <DetailList className="max-w-lg" rows={[
      [t("Nominal pengajuan"), money(preview.amount)],
      [t("Tenor pinjaman"), preview.tenor_months + " " + t("bulan")],
      [t("Bunga pinjaman"), preview.annual_rate_bps / 100 + "% p.a."],
      [t("Versi tarif"), "#" + preview.policy_id],
      [t("Sumber tarif"), t(preview.base_rate_fallback === undefined ? "Tarif persetujuan" : preview.base_rate_fallback ? "Tarif global" : "Rentang nominal")],
      [t("Angsuran reguler"), money(preview.regular_installment_amount)],
      [t("Total bunga"), money(preview.total_interest)],
      [t("Total pembayaran"), money(preview.total_payable)],
    ]}/>}
    <LocalRows resetKey={preview.amount+":"+preview.tenor_months+":"+preview.annual_rate_bps+":"+preview.policy_id} rows={preview.installments} label={t("Jadwal angsuran")} loading={loading} stale={stale} statusText={statusText}>
      {rows=><ScheduleTable rows={rows}/>}
    </LocalRows>
    </section>;
}
export function Allocations({ rows }: {
    rows: Allocation[];
}) {
    const { t, money } = usePreferences();
    if (rows.length === 1) {
        const row = rows[0];
        return <DetailList className="max-w-lg" rows={[
            [t("Angsuran / referensi jadwal"), row.installment_number ?? "#" + row.schedule_id],
            [t("Pokok"), money(row.principal_amount)],
            [t("Bunga"), money(row.interest_amount)],
        ]}/>;
    }
    return <LocalRows rows={rows} label={t("Alokasi pembayaran")} printAll>{visible=><table className="w-full text-left text-sm">
    <caption className="text-left">{t("Alokasi pembayaran")}</caption>
    <thead>
    <tr>
    <th className="p-2">{t("Angsuran / referensi jadwal")}</th>
    <th className="p-2">{t("Pokok")}</th>
    <th className="p-2">{t("Bunga")}</th>
    </tr>
    </thead>
    <tbody>{visible.map(r => <tr key={r.schedule_id}>
        <td className="p-2">{r.installment_number ?? "#" + r.schedule_id}</td>
        <td className="p-2">{money(r.principal_amount)}</td>
        <td className="p-2">{money(r.interest_amount)}</td>
        </tr>)}</tbody>
    </table>}</LocalRows>;
}
export type Execute = (path: string, input: unknown) => Promise<void>;
