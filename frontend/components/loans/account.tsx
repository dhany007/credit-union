"use client";
import { useState } from "react";
import { api } from "@/lib/api";
import { repaymentBody, type Account, type Installment, type Payment, type Version, type PaymentPreview, type Staff, type Receipt } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Allocations, ErrorNotice, Field, Paged, ScheduleTable, type Execute } from "./shared";
export function AccountDetail({ account: a, user, execute, onReceipt }: {
    account: Account;
    user: Staff;
    execute: Execute;
    onReceipt: (receipt: Receipt) => void;
}) {
    const { t, money, locale } = usePreferences();
    const [section, setSection] = useState<"summary" | "schedule" | "payments" | "versions">("summary");
    const [version, setVersion] = useState(a.current_schedule_version_id);
    const [amount, setAmount] = useState("");
    const [note, setNote] = useState("");
    const [preview, setPreview] = useState<PaymentPreview | null>(null);
    const [confirm, setConfirm] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const root = "/loan-accounts/" + a.id;
    async function review(e: React.FormEvent) {
        e.preventDefault();
        setLoading(true);
        setPreview(null);
        setConfirm(false);
        setError("");
        try {
            setPreview(await api<PaymentPreview>(root + "/repayment-preview", { method: "POST", body: JSON.stringify({ amount: Number(amount) }) }));
        }
        catch (e) {
            setError(e instanceof Error ? e.message : "Failed to fetch");
        }
        finally {
            setLoading(false);
        }
    }
    const formatDate = (value: string) => new Date(value).toLocaleString(locale === "id" ? "id-ID" : "en-US", { timeZone: "Asia/Jakarta" });
    return <article className="space-y-5 rounded-lg border p-5">
    <h3 className="font-semibold">{a.loan_number} · {a.member_name}</h3>
    <p>{a.account_number} · {a.responsible_branch} · {t(a.status === "paid_off" ? "Lunas" : "Aktif")}</p>
    <nav className="flex flex-wrap gap-2" aria-label={t("Rincian pinjaman")}>{(["summary", "schedule", "payments", "versions"] as const).map((v, i) => <Button key={v} variant={section === v ? "default" : "outline"} aria-pressed={section === v} onClick={() => setSection(v)}>{[t("Ringkasan"), t("Jadwal angsuran"), t("Pembayaran"), t("Riwayat versi")][i]}</Button>)}</nav>
 {section === "summary" && <dl className="grid gap-3 sm:grid-cols-2">
        <dt>{t("Pokok dicairkan")}</dt>
        <dd>{money(a.principal_amount)}</dd>
        <dt>{t("Sisa pokok aktual")}</dt>
        <dd>{money(a.actual_principal_outstanding)}</dd>
        <dt>{t("Bunga pinjaman")}</dt>
        <dd>{a.annual_rate_bps / 100}% p.a. · {t("Anuitas tetap")}</dd>
        <dt>{t("Tenor yang diajukan (bulan)")}</dt>
        <dd>{a.tenor_months}</dd>
        <dt>{t("Tanggal pencairan")}</dt>
        <dd>{formatDate(a.disbursed_at)} WIB</dd>
        <dt>{t("Pengajuan")}</dt>
        <dd>PJ-{a.application_id}</dd><dt>{t("Bukti pencairan")}</dt><dd><Button variant="outline" onClick={()=>onReceipt({loan_id:a.id,loan_number:a.loan_number,reference:a.disbursement_transaction_id,amount:a.principal_amount,disbursed_at:a.disbursed_at,actor:"#"+a.created_by,branch:"KCP #"+a.disbursing_branch_id})}>{t("Lihat bukti pencairan")}</Button></dd>
        </dl>}
 {section === "schedule" && <>
        <p>{t("Versi jadwal")} #{version} <Button variant="ghost" onClick={() => setVersion(a.current_schedule_version_id)}>{t("Versi saat ini")}</Button>
        </p>
        <Paged<Installment> key={version} path={root + "/installments?schedule_version_id=" + version} caption={t("Jadwal angsuran")}>{rows => <ScheduleTable rows={rows} actual/>}</Paged>
        </>}
 {section === "versions" && <Paged<Version> path={root + "/schedule-versions"} caption={t("Riwayat versi")}>{rows => <div className="space-y-4">{rows.map(v => <section key={v.id} className="rounded border p-4 space-y-2">
                <p>#{v.version_number} · {t(v.version_type === "initial" ? "Awal" : "Restrukturisasi")} · {v.is_current_version ? t("Versi saat ini") : t("Historis")}</p>
                <p>{v.effective_date.slice(0, 10)} · {v.interest_rate_bps / 100}% p.a. · {v.tenor_months} {t("bulan")}</p>
                <p>{v.reason} · {t("Disetujui oleh")} #{v.approved_by} · {formatDate(v.approved_at)} WIB</p>
                <Button variant="outline" onClick={() => { setVersion(v.id); setSection("schedule"); }}>{t("Lihat jadwal")}</Button>
                </section>)}</div>}</Paged>}
 {section === "payments" && <div className="space-y-5">{user.role === "teller" && a.status === "active" && <form onSubmit={review} className="space-y-4">
            <fieldset disabled={loading} className="grid gap-4 sm:grid-cols-2">
            <legend className="mb-3 font-semibold">{t("Catat pembayaran")}</legend>
            <Field label={t("Nominal pembayaran (Rp)")}>
            <Input required type="number" min={1} max={1000000000000} step={1} value={amount} onChange={e => { setAmount(e.target.value); setPreview(null); setConfirm(false); }}/>
            </Field>
            <Field label={t("Catatan (opsional)")}>
            <Textarea maxLength={500} value={note} onChange={e => { setNote(e.target.value); setConfirm(false); }}/>
            </Field>
            <Button>{t("Tinjau pembayaran")}</Button>
            </fieldset>
            <p className="text-sm text-muted-foreground">{t("Pembayaran dialokasikan ke jadwal tertua: bunga lalu pokok. Kelebihan pembayaran ditolak.")}</p>
            </form>}
 <ErrorNotice error={error}/>{preview && <section className="space-y-3 rounded border p-4">
            <p>{t("Sisa pokok aktual")}: {money(preview.principal_before)} → {money(preview.principal_after)}</p>
            <p>{t("Sisa kewajiban terjadwal")}: {money(preview.outstanding_payable)}</p>
            <Allocations rows={preview.allocations}/>{!confirm ? <Button onClick={() => setConfirm(true)}>{t("Lanjutkan pembayaran")}</Button> : <section role="region" aria-label={t("Konfirmasi pinjaman")} tabIndex={-1} ref={node => node?.focus()} className="space-y-3">
                <p>{a.member_name} · {a.loan_number} · {money(preview.amount)} · {user.branch_name}</p>
                <Button onClick={() => void execute(root + "/repayments", repaymentBody(preview, note))}>{t("Konfirmasi pembayaran")}</Button>
                <Button variant="ghost" onClick={() => setConfirm(false)}>{t("Batal")}</Button>
                </section>}</section>}
 <Paged<Payment> path={root + "/repayments"} caption={t("Riwayat pembayaran")}>{rows => <div className="space-y-4">{rows.map(r => <section key={r.id} className="rounded border p-4 space-y-2">
                <h4 className="font-semibold">{r.repayment_number} · {money(r.amount)}</h4>
                <p>{formatDate(r.paid_at)} WIB · {r.branch} · {r.actor}</p>
                <p>{t("Transaksi")} #{r.transaction_id} · {r.note}</p>
                <Allocations rows={r.allocations}/>
                <Button variant="outline" onClick={() => onReceipt({ loan_id: a.id, loan_number: a.loan_number, repayment_number: r.repayment_number, reference: r.transaction_id, amount: r.amount, paid_at: r.paid_at, actor: r.actor, branch: r.branch, allocations: r.allocations })}>{t("Lihat bukti pembayaran")}</Button>
                </section>)}</div>}</Paged>
        </div>}
 </article>;
}
