"use client";
import { Fragment, useState } from "react";
import { useLoanPreview } from "./use-loan-preview";
import { jakartaToday, payoutBody, type Application, type Installment, type Staff } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { AmountInput } from "@/components/amount-input";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Field, ErrorNotice, SchedulePreview, ScheduleTable, Paged, type Execute } from "./shared";
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
    const [confirm, setConfirm] = useState<"approved" | "rejected" | "payout" | null>(null);
    const canDecide = user.role === "supervisor" && (a.status === "submitted" || (a.status === "approved" && !a.pricing_complete && !a.linked_loan_id));
    const payoutPreview = payout && user.role === "teller" && a.status === "approved" && !a.linked_loan_id;
    const estimatePreview = !payout && (a.status === "submitted" || canDecide);
    const validTerms = amount !== "" && tenor !== "" && Number.isSafeInteger(Number(amount)) &&
      Number(amount) >= 1 && Number(amount) <= 1000000000000 && Number.isInteger(Number(tenor)) &&
      Number(tenor) >= 1 && Number(tenor) <= 120;
    const { preview, error, loading, retry } = useLoanPreview(
      payout ? "/loan-applications/" + a.id + "/disbursement-preview" : "/loans/preview",
      payoutPreview ? { expected_approval_revision: a.approval_revision } :
        estimatePreview && validTerms ? { amount: Number(amount), tenor_months: Number(tenor), expected_disbursement_date: jakartaToday() } : null,
    );
    function invalidate() { setConfirm(null); }
    return <article className="rounded-lg border p-5">
    <fieldset aria-busy={loading} className="space-y-5 min-w-0">
    <section className="space-y-4" aria-label={t("Ringkasan pengajuan")}>
      <h3 className="font-semibold">{t("Ringkasan pengajuan")}</h3>
      <dl className="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)] gap-x-3 gap-y-3 text-sm sm:grid-cols-[12rem_minmax(0,1fr)]">
        {[
          [t("Nomor pengajuan"), a.application_number],
          [t("Anggota"), a.member_name],
          [t("Nomor rekening"), a.account_number],
          [t("Nominal pengajuan"), money(a.amount)],
          [t("Tenor pinjaman"), a.tenor_months + " " + t("bulan")],
          [t("KCP penerima"), a.receiving_branch],
          [t("KCP penanggung jawab"), a.responsible_branch],
          [t("Tujuan pinjaman"), a.purpose],
          [t("Risiko"), t(a.risk === "high" ? "Tinggi" : a.risk === "medium" ? "Sedang" : "Rendah")],
          ...(canDecide && !payout ? [[t("Batas kewenangan"), money(limit)]] : []),
          ...(a.approved_amount !== null ? [[t("Syarat disetujui"), money(a.approved_amount) + " · " + a.approved_tenor_months + " " + t("bulan") + " · " + (a.approved_interest_rate_bps??0)/100 + "% p.a."]] : []),
          ...(a.decision_reason ? [[t("Alasan keputusan"), a.decision_reason]] : []),
        ].map(([label, value]) => <Fragment key={label}>
          <dt className="break-words text-muted-foreground">{label}</dt>
          <dd className="grid min-w-0 grid-cols-[0.5rem_minmax(0,1fr)] gap-x-2">
            <span aria-hidden="true">:</span><span className="break-words font-medium">{value}</span>
          </dd>
        </Fragment>)}
      </dl>
    </section>
 {canDecide && !payout && <>
        <div className="grid gap-4 sm:grid-cols-2">
        <Field label={t("Pokok disetujui (Rp)")}>
        <AmountInput required value={amount} onValueChange={digits => { setAmount(digits); invalidate(); }}/>
        </Field>
        <Field label={t("Tenor yang diajukan (bulan)")}>
        <Input type="number" min={1} max={120} step={1} value={tenor} onChange={e => { setTenor(e.target.value); invalidate(); }}/>
        </Field>
        </div>
        </>}
 {loading && <p role="status" className="text-sm text-muted-foreground">{t("Menghitung jadwal angsuran...")}</p>}
 {(estimatePreview && !validTerms) && <p className="text-sm text-muted-foreground">{t("Isi nominal dan tenor yang valid untuk melihat jadwal.")}</p>}
 <ErrorNotice error={error}/>
 {error && <Button type="button" variant="outline" onClick={()=>{setConfirm(null);retry();}}>{t("Muat ulang jadwal")}</Button>}
 {preview && <SchedulePreview key={JSON.stringify(preview)} preview={preview}/>}
 {a.linked_loan_id && <section className="space-y-3">
   <h3 className="font-semibold">{t("Jadwal angsuran")}</h3>
   <Paged<Installment> path={"/loan-accounts/" + a.linked_loan_id + "/installments"} caption={t("Jadwal angsuran")}>{rows=><ScheduleTable rows={rows} actual/>}</Paged>
 </section>}
 {a.status === "approved" && !a.linked_loan_id && !canDecide && !payout && <p className="text-sm text-muted-foreground">{t("Jadwal kontrak tersedia setelah pencairan.")}</p>}
 {canDecide && !payout && <form className="space-y-4" onSubmit={e => { e.preventDefault(); setConfirm("approved"); }}>
        <Field label={t("Alasan keputusan")}>
        <Textarea required minLength={5} maxLength={1000} value={reason} onChange={e => { setReason(e.target.value); setConfirm(null); }}/>
        </Field>
        <div className="flex gap-3">
        <Button disabled={loading || !preview || limit <= 0 || Number(amount) > limit || Number(amount) > a.amount || Number(tenor) > a.tenor_months || !validTerms}>{t("Setujui")}</Button>{a.status === "submitted" && <Button type="button" variant="outline" disabled={reason.trim().length < 5} onClick={() => setConfirm("rejected")}>{t("Tolak")}</Button>}</div>
        </form>}
 {payout && preview && user.role === "teller" && <form className="space-y-3" onSubmit={e => { e.preventDefault(); setConfirm("payout"); }}>
        <p>{t("Kas tersedia")}: {money(preview.available_cash ?? 0)} · {t(preview.payout_eligible ? "Layak dicairkan" : "Tidak layak dicairkan")}</p>{(["Penerima sudah diverifikasi", "Perjanjian sudah ditandatangani", "Persyaratan sudah dipenuhi"] as const).map(label => <label key={label} className="flex gap-2 items-center">
            <input type="checkbox" required onChange={() => setConfirm(null)}/>{t(label)}</label>)}<Field label={t("Catatan verifikasi")}>
        <Textarea required minLength={5} maxLength={500} value={reason} onChange={e => { setReason(e.target.value); setConfirm(null); }}/>
        </Field>
        <Button disabled={!preview.payout_eligible}>{t("Tinjau pencairan")}</Button>
        </form>}
 {confirm && (confirm === "rejected" || preview) && <section role="region" aria-label={t("Konfirmasi pinjaman")} tabIndex={-1} ref={node => node?.focus()} className="space-y-3 rounded-lg border border-primary p-4">
        <h4 className="font-semibold">{t("Konfirmasi pinjaman")}</h4>
        <p>{a.member_name} · {a.account_number} · {money(confirm === "rejected" ? a.amount : Number(amount))} · {user.branch_name}</p>
        <p>{t(confirm === "payout" ? "Konfirmasi setelah uang tunai siap diserahkan. Jangan ulang dengan referensi baru jika respons terputus." : confirm === "approved" ? "Setujui syarat sesuai preview yang ditinjau?" : "Tolak pengajuan ini?")}</p>
        <Button disabled={confirm !== "rejected" && (loading || !preview)} onClick={() => {
                if (confirm !== "rejected" && !preview) return;
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
