"use client";
import { useRef, useState } from "react";
import { api } from "@/lib/api";
import { jakartaToday, payoutBody, type Application, type Preview, type Staff, type Branch } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { AmountInput } from "@/components/amount-input";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Field, ErrorNotice, SchedulePreview, selectClass, type Execute } from "./shared";
type Member = {
    id: number;
    name: string;
    account_number: string;
};
export function ApplicationForm({ user, branches, execute }: {
    user: Staff;
    branches: Branch[];
    execute: Execute;
}) {
    const { t } = usePreferences();
    const [account, setAccount] = useState("");
    const [requestedAmount, setRequestedAmount] = useState("");
    const [member, setMember] = useState<Member | null>(null);
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const lookup = useRef(0);
    const estimateVersion = useRef(0);
    const [estimate, setEstimate] = useState<Preview | null>(null);
    const [estimating, setEstimating] = useState(false);
    async function find(e: React.FormEvent) {
        e.preventDefault();
        const version = ++lookup.current;
        setLoading(true);
        setError("");
        try {
            const m = await api<Member>("/members/by-account/" + account);
            if (version === lookup.current)
                setMember(m);
        }
        catch (e) {
            if (version === lookup.current)
                setError(e instanceof Error ? e.message : "Failed to fetch");
        }
        finally {
            if (version === lookup.current)
                setLoading(false);
        }
    }
    return <section className="space-y-4 rounded-lg border p-5">
    <h3 className="font-semibold">{t("Pengajuan pinjaman reguler")}</h3>
    <form onSubmit={find} className="flex flex-wrap items-end gap-3">
    <Field label={t("Nomor rekening")}>
    <Input required pattern="[0-9]{10}" maxLength={10} value={account} onChange={e => { lookup.current++; estimateVersion.current++; setEstimate(null); setAccount(e.target.value); setMember(null); setLoading(false); }}/>
    </Field>
    <Button disabled={loading}>{t("Cari rekening")}</Button>
    </form>
    <ErrorNotice error={error}/>{member && <form className="grid gap-4 md:grid-cols-2" onChange={() => { estimateVersion.current++; setEstimate(null); }} onSubmit={e => { e.preventDefault(); const d = Object.fromEntries(new FormData(e.currentTarget)); void execute("/loans", { member_id: member.id, amount: Number(d.amount), tenor_months: Number(d.tenor), responsible_branch_id: Number(d.branch), purpose: d.purpose, risk: d.risk }); }}>
        <p className="md:col-span-2">{member.name} · {member.account_number}</p>
        <Field label={t("Nominal pengajuan (Rp)")}>
        <AmountInput required name="amount" value={requestedAmount} onValueChange={setRequestedAmount}/>
        </Field>
        <Field label={t("Tenor yang diajukan (bulan)")}>
        <Input required name="tenor" type="number" min={1} max={120} step={1} defaultValue={12}/>
        </Field>
        <Field label={t("KCP penanggung jawab")}>
        <select name="branch" className={selectClass} defaultValue={user.branch_id}>{branches.map(b => <option key={b.id} value={b.id}>{b.name}</option>)}</select>
        </Field>
        <Field label={t("Risiko awal untuk diverifikasi Supervisor")}>
        <select name="risk" className={selectClass}>
        <option value="low">{t("Rendah")}</option>
        <option value="medium">{t("Sedang")}</option>
        <option value="high">{t("Tinggi")}</option>
        </select>
        </Field>
        <Field label={t("Tujuan pinjaman")}>
        <Textarea required name="purpose" minLength={5} maxLength={1000}/>
        </Field>
        <div className="md:col-span-2">
        <Button type="button" variant="outline" disabled={estimating} onClick={async (e) => {
                const form = e.currentTarget.form;
                if (!form)
                    return;
                const d = new FormData(form);
                const version = ++estimateVersion.current;
                setEstimating(true);
                setError("");
                setEstimate(null);
                try {
                    const result = await api<Preview>("/loans/preview", { method: "POST", body: JSON.stringify({ amount: Number(d.get("amount")), tenor_months: Number(d.get("tenor")), expected_disbursement_date: jakartaToday() }) });
                    if (version === estimateVersion.current)
                        setEstimate(result);
                }
                catch (e) {
                    if (version === estimateVersion.current)
                        setError(e instanceof Error ? e.message : "Failed to fetch");
                }
                finally {
                    setEstimating(false);
                }
            }}>{t("Preview anuitas")}</Button>
 <Button>{t("Kirim pengajuan")}</Button>
        </div>
        </form>}{estimate && <SchedulePreview key={JSON.stringify(estimate)} preview={estimate}/>}</section>;
}
export function ApplicationDetail({ application: a, user, limit, payout, execute }: {
    application: Application;
    user: Staff;
    limit: number;
    payout: boolean;
    execute: Execute;
}) {
    const { t, money } = usePreferences();
    const [amount, setAmount] = useState(String(a.approved_amount ?? a.amount));
    const [tenor, setTenor] = useState(String(a.approved_tenor_months ?? a.tenor_months));
    const [reason, setReason] = useState("");
    const [preview, setPreview] = useState<Preview | null>(null);
    const [error, setError] = useState("");
    const [loading, setLoading] = useState(false);
    const [confirm, setConfirm] = useState<"approved" | "rejected" | "payout" | null>(null);
    const canDecide = user.role === "supervisor" && (a.status === "submitted" || (a.status === "approved" && !a.pricing_complete && !a.linked_loan_id));
    async function calculate() {
        setLoading(true);
        setError("");
        setPreview(null);
        setConfirm(null);
        try {
            setPreview(await api<Preview>(payout ? "/loan-applications/" + a.id + "/disbursement-preview" : "/loans/preview", { method: "POST", body: JSON.stringify(payout ? { expected_approval_revision: a.approval_revision } : { amount: Number(amount), tenor_months: Number(tenor), expected_disbursement_date: jakartaToday() }) }));
        }
        catch (e) {
            setError(e instanceof Error ? e.message : "Failed to fetch");
        }
        finally {
            setLoading(false);
        }
    }
    function invalidate() { setPreview(null); setConfirm(null); setError(""); }
    return <article className="rounded-lg border p-5">
    <fieldset disabled={loading} className="space-y-5 min-w-0">
    <h3 className="font-semibold">{a.application_number} · {a.member_name}</h3>
    <p>{a.account_number} · {money(a.amount)} · {a.tenor_months} {t("bulan")}</p>
    <p>{t("KCP penerima")}: {a.receiving_branch} · {t("KCP penanggung jawab")}: {a.responsible_branch}</p>
    <p>{t("Tujuan pinjaman")}: {a.purpose}</p>{a.approved_amount !== null && <p>{t("Syarat disetujui")}: {money(a.approved_amount)} · {a.approved_tenor_months} {t("bulan")} · {(a.approved_interest_rate_bps??0)/100}% p.a.</p>}
    <p>{t("Risiko")}: {t(a.risk === "high" ? "Tinggi" : a.risk === "medium" ? "Sedang" : "Rendah")}</p>{a.decision_reason && <p>{t("Alasan keputusan")}: {a.decision_reason}</p>}
 {canDecide && !payout && <>
        <p className="text-sm text-muted-foreground">{t("Batas kewenangan")}: {money(limit)}</p>
        <div className="grid gap-4 sm:grid-cols-2">
        <Field label={t("Pokok disetujui (Rp)")}>
        <AmountInput required value={amount} onValueChange={digits => { setAmount(digits); invalidate(); }}/>
        </Field>
        <Field label={t("Tenor yang diajukan (bulan)")}>
        <Input type="number" min={1} max={120} step={1} value={tenor} onChange={e => { setTenor(e.target.value); invalidate(); }}/>
        </Field>
        </div>
        </>}
 {(canDecide && !payout || payout && user.role === "teller" && !a.linked_loan_id) && <Button type="button" variant="outline" disabled={loading} onClick={calculate}>{t(payout ? "Periksa kelayakan pencairan" : "Preview anuitas")}</Button>}
 <ErrorNotice error={error}/>{preview && <SchedulePreview key={JSON.stringify(preview)} preview={preview}/>}
 {canDecide && !payout && <form className="space-y-4" onSubmit={e => { e.preventDefault(); setConfirm("approved"); }}>
        <Field label={t("Alasan keputusan")}>
        <Textarea required minLength={5} maxLength={1000} value={reason} onChange={e => { setReason(e.target.value); setConfirm(null); }}/>
        </Field>
        <div className="flex gap-3">
        <Button disabled={!preview || limit <= 0 || Number(amount) > limit || !Number.isSafeInteger(Number(amount))}>{t("Setujui")}</Button>{a.status === "submitted" && <Button type="button" variant="outline" disabled={reason.trim().length < 5} onClick={() => setConfirm("rejected")}>{t("Tolak")}</Button>}</div>
        </form>}
 {payout && preview && user.role === "teller" && <form className="space-y-3" onSubmit={e => { e.preventDefault(); setConfirm("payout"); }}>
        <p>{t("Kas tersedia")}: {money(preview.available_cash ?? 0)} · {t(preview.payout_eligible ? "Layak dicairkan" : "Tidak layak dicairkan")}</p>{(["Penerima sudah diverifikasi", "Perjanjian sudah ditandatangani", "Persyaratan sudah dipenuhi"] as const).map(label => <label key={label} className="flex gap-2 items-center">
            <input type="checkbox" required onChange={() => setConfirm(null)}/>{t(label)}</label>)}<Field label={t("Catatan verifikasi")}>
        <Textarea required minLength={5} maxLength={500} value={reason} onChange={e => { setReason(e.target.value); setConfirm(null); }}/>
        </Field>
        <Button disabled={!preview.payout_eligible}>{t("Tinjau pencairan")}</Button>
        </form>}
 {confirm && <section role="region" aria-label={t("Konfirmasi pinjaman")} tabIndex={-1} ref={node => node?.focus()} className="space-y-3 rounded-lg border border-primary p-4">
        <h4 className="font-semibold">{t("Konfirmasi pinjaman")}</h4>
        <p>{a.member_name} · {a.account_number} · {money(confirm === "rejected" ? a.amount : Number(amount))} · {user.branch_name}</p>
        <p>{t(confirm === "payout" ? "Konfirmasi setelah uang tunai siap diserahkan. Jangan ulang dengan referensi baru jika respons terputus." : confirm === "approved" ? "Setujui syarat sesuai preview yang ditinjau?" : "Tolak pengajuan ini?")}</p>
        <Button onClick={() => {
                if (confirm === "payout" && preview)
                    void execute("/loan-applications/" + a.id + "/disburse", payoutBody(preview, reason));
                else
                    void execute("/loans/" + a.id + "/decision", { decision: confirm, reason, ...(confirm === "approved" && preview ? { approved_amount: preview.amount, approved_tenor_months: preview.tenor_months, expected_policy_id: preview.policy_id, expected_rate_bps: preview.annual_rate_bps, expected_approval_revision: a.approval_revision } : {}) });
            }}>{t("Konfirmasi")}</Button>
        <Button variant="ghost" onClick={() => setConfirm(null)}>{t("Batal")}</Button>
        </section>}
 </fieldset>
    </article>;
}
