"use client";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
export { ErrorNotice } from "@/components/modal";
import { ErrorNotice } from "@/components/modal";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import type { Page, Installment, Preview, Allocation } from "@/lib/loans";
export const selectClass = "h-10 rounded-md border bg-card px-3 text-sm w-full";
export function Field({ label, children }: {
    label: string;
    children: React.ReactNode;
}) {
    return <label className="grid gap-2 text-sm">
    <span className="font-medium">{label}</span>{children}</label>;
}
export function useRead<T>(path: string) {
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
    }, [path]);
    return state.path === path ? state : { path };
}
export function Paged<T>({ path, children, caption }: {
    path: string;
    children: (items: T[]) => React.ReactNode;
    caption: string;
}) {
    const { t } = usePreferences();
    const [cursors, setCursors] = useState([""]);
    const cursor = cursors[cursors.length - 1];
    const { data, error } = useRead<Page<T>>(path + (path.includes("?") ? "&" : "?") + "cursor=" + encodeURIComponent(cursor));
    return <section className="space-y-3" aria-label={caption}>
    <ErrorNotice error={error ?? ""}/>{!data && !error ? <p role="status">{t("Memuat...")}</p> : data ? <>{data.items.length ? children(data.items) : <p className="p-6 text-muted-foreground">{t("Tidak ada data pinjaman.")}</p>}<div className="flex flex-wrap items-center gap-3">
        <Button variant="outline" disabled={cursors.length === 1} onClick={() => setCursors(cursors.slice(0, -1))}>{t("Sebelumnya")}</Button>
        <span className="text-sm">{t("Halaman pinjaman {page} · maksimal 20 baris", { page: cursors.length })}</span>
        <Button variant="outline" disabled={!data.next_cursor} onClick={() => setCursors([...cursors, data.next_cursor])}>{t("Berikutnya")}</Button>
        </div>
        </> : null}</section>;
}
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
export function SchedulePreview({ preview }: {
    preview: Preview;
}) {
    const { t, money } = usePreferences();
    const [page, setPage] = useState(0);
    return <section className="space-y-3 rounded-lg border p-4">
    <h3 className="font-semibold">{t("Estimasi jadwal angsuran")}</h3>
    <p className="text-sm text-muted-foreground">{t("Estimasi jadwal; tanggal aktual dan pembulatan final ditentukan saat pencairan.")}</p>{preview.base_rate_fallback !== undefined && <p className="text-sm">{t(preview.base_rate_fallback ? "Menggunakan tarif global" : "Menggunakan tarif rentang nominal")}</p>}
    <p>{money(preview.amount)} · {preview.tenor_months} {t("bulan")} · {preview.annual_rate_bps / 100}% p.a. · {t("Versi tarif")} #{preview.policy_id}</p>
    <p className="text-sm">{t("Angsuran reguler")}: {money(preview.regular_installment_amount)} · {t("Total bunga")}: {money(preview.total_interest)} · {t("Total pembayaran")}: {money(preview.total_payable)}</p>
    <ScheduleTable rows={preview.installments.slice(page * 20, (page + 1) * 20)}/>
    <div className="flex gap-2">
    <Button type="button" variant="outline" disabled={!page} onClick={() => setPage(page - 1)}>{t("Sebelumnya")}</Button>
    <Button type="button" variant="outline" disabled={(page + 1) * 20 >= preview.installments.length} onClick={() => setPage(page + 1)}>{t("Berikutnya")}</Button>
    </div>
    </section>;
}
export function Allocations({ rows }: {
    rows: Allocation[];
}) {
    const { t, money } = usePreferences();
    return <div className="max-h-72 overflow-auto">
    <table className="w-full text-left text-sm">
    <caption className="text-left">{t("Alokasi pembayaran")}</caption>
    <thead>
    <tr>
    <th className="p-2">{t("Angsuran / referensi jadwal")}</th>
    <th className="p-2">{t("Pokok")}</th>
    <th className="p-2">{t("Bunga")}</th>
    </tr>
    </thead>
    <tbody>{rows.map(r => <tr key={r.schedule_id}>
        <td className="p-2">{r.installment_number ?? "#" + r.schedule_id}</td>
        <td className="p-2">{money(r.principal_amount)}</td>
        <td className="p-2">{money(r.interest_amount)}</td>
        </tr>)}</tbody>
    </table>
    </div>;
}
export type Execute = (path: string, input: unknown) => Promise<void>;
