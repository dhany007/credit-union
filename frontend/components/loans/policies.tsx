"use client";
import { useState } from "react";
import { usePreferences } from "@/components/preferences";
import { nextJakartaDate, rateToBps } from "@/lib/interest";
import { Button } from "@/components/ui/button";
import { AmountInput } from "@/components/amount-input";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ErrorNotice, Field, Paged, useRead, type Execute } from "./shared";
import type { Page } from "@/lib/loans";
type Policy = {
    id: number;
    effective_date: string;
    base_rate_bps: number;
    bands: {
        min_amount: number;
        max_amount: number | null;
        rate_bps: number;
    }[];
    reason: string;
};
export function LoanPolicies({ execute }: {
    execute: Execute;
}) {
    const { t, money } = usePreferences();
    const [attempt, setAttempt] = useState(0);
    const { data, error } = useRead<Page<Policy>>("/loan-interest/policies", attempt);
    const [bands, setBands] = useState<{
        min: string;
        max: string;
        rate: string;
    }[]>([]);
    const [invalid, setInvalid] = useState("");
    return <section className="space-y-5 rounded border p-5">
    <h3 className="font-semibold">{t("Tarif bunga pinjaman")}</h3>
    <p className="text-sm text-muted-foreground">{t("Tarif minimal 3% p.a. untuk seluruh pokok sesuai rentang nominal. Berlaku paling cepat besok; pinjaman lama tidak berubah.")}</p>
    {error && <div className="flex items-center gap-3"><p role="status">{t("Gagal memuat data. Silakan coba lagi.")}</p><Button type="button" variant="outline" onClick={() => setAttempt(value => value + 1)}>{t("Muat ulang")}</Button></div>}
    <ErrorNotice error={invalid} onClose={()=>setInvalid("")}/>
    <form className="space-y-4" onSubmit={e => {
            e.preventDefault();
            const d = Object.fromEntries(new FormData(e.currentTarget));
            const base = rateToBps(String(d.base));
            const parsed = bands.map(b => ({ min_amount: Number(b.min), max_amount: b.max === "" ? null : Number(b.max), rate_bps: rateToBps(b.rate) }));
            if (base === null || base < 300 || parsed.some((b, i) => bands[i].min.trim() === "" || !Number.isSafeInteger(b.min_amount) || b.min_amount < 0 || b.rate_bps === null || b.rate_bps < 300 || (b.max_amount !== null && (!Number.isSafeInteger(b.max_amount) || b.max_amount <= b.min_amount)))) {
                setInvalid("Invalid loan interest policy");
                return;
            }
            setInvalid("");
            void execute("/loan-interest/policies", { effective_date: d.effective, base_rate_bps: base, bands: parsed, reason: d.reason, expected_latest_id: data?.items[0]?.id ?? null });
        }}>
    <div className="grid gap-4 sm:grid-cols-2">
    <Field label={t("Tanggal berlaku")}>
    <Input name="effective" required type="date" min={nextJakartaDate()} defaultValue={nextJakartaDate()}/>
    </Field>
    <Field label={t("Tarif global (% p.a.)")}>
    <Input name="base" required type="number" min={3} max={100} step=".01" placeholder="3"/>
    </Field>
    </div>
    <p className="text-sm">{t("Rentang nominal menggunakan batas bawah termasuk dan batas atas tidak termasuk. Di luar rentang memakai tarif global.")}</p>{bands.map((b, i) => <fieldset key={i} className="grid gap-3 rounded border p-3 sm:grid-cols-4">
        <legend>{t("Rentang nominal")} {i + 1}</legend>
        <Field label={t("Nominal mulai (Rp)")}>
        <AmountInput required minimum={0} value={b.min} onValueChange={digits => setBands(bands.map((x, j) => j === i ? { ...x, min: digits } : x))}/>
        </Field>
        <Field label={t("Nominal hingga (Rp)")}>
        <AmountInput placeholder={t("Tanpa batas")} value={b.max} onValueChange={digits => setBands(bands.map((x, j) => j === i ? { ...x, max: digits } : x))}/>
        </Field>
        <Field label={t("Bunga (% p.a.)")}>
        <Input required type="number" min={3} max={100} step=".01" value={b.rate} onChange={e => setBands(bands.map((x, j) => j === i ? { ...x, rate: e.target.value } : x))}/>
        </Field>
        <Button type="button" variant="outline" onClick={() => setBands(bands.filter((_, j) => i !== j))}>{t("Hapus")}</Button>
        </fieldset>)}<Button type="button" variant="outline" disabled={bands.length >= 100} onClick={() => setBands([...bands, { min: "", max: "", rate: "" }])}>{t("Tambah rentang")}</Button>
    <Field label={t("Alasan perubahan")}>
    <Textarea required name="reason" minLength={5} maxLength={500}/>
    </Field>
    <Button disabled={!data}>{t("Simpan versi tarif")}</Button>
    </form>
    <Paged<Policy> path="/loan-interest/policies" caption={t("Riwayat tarif")}>{rows => <div className="space-y-3">{rows.map(p => <section key={p.id} className="rounded border p-3 space-y-2">
            <p>#{p.id} · {p.effective_date.slice(0, 10)} · {p.base_rate_bps / 100}% p.a.</p>
            <p>{p.reason}</p>{p.bands.map((b, i) => <p key={i}>{money(b.min_amount)} – {b.max_amount === null ? t("Tanpa batas") : money(b.max_amount)} · {b.rate_bps / 100}% p.a.</p>)}</section>)}</div>}</Paged>
    </section>;
}
