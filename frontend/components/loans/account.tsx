"use client";
import { Modal } from "@/components/modal";
import { DetailList } from "@/components/ui/detail-list";
import { useState } from "react";
import { api } from "@/lib/api";
import { formatLoanTimestamp, repaymentBody, type Account, type Installment, type Payment, type Version, type PaymentPreview, type Staff, type Receipt } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
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
    const [note, setNote] = useState("");
    const [preview, setPreview] = useState<PaymentPreview | null>(null);
    const [confirm, setConfirm] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState("");
    const installment = preview?.allocations.length === 1 ? preview.allocations[0] : undefined;
    const root = "/loan-accounts/" + a.id;
    const scheduleVersion = section === "summary" ? a.current_schedule_version_id : version;
    async function review(e: React.FormEvent) {
        e.preventDefault();
        setLoading(true);
        setPreview(null);
        setConfirm(false);
        setError("");
        try {
            setPreview(await api<PaymentPreview>(root + "/repayment-preview", { method: "POST", body: JSON.stringify({ next_installment: true }) }));
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
 {section === "summary" && <DetailList className="max-w-lg" rows={[
        [t("Pokok dicairkan"), money(a.principal_amount)],
        [t("Sisa pokok aktual"), money(a.actual_principal_outstanding)],
        [t("Bunga pinjaman"), a.annual_rate_bps / 100 + "% p.a. · " + t("Anuitas tetap")],
        [t("Tenor yang diajukan (bulan)"), a.tenor_months],
        [t("Tanggal pencairan"), formatDate(a.disbursed_at) + " WIB"],
        [t("Pengajuan"), "PJ-" + a.application_id],
        [t("Bukti pencairan"), <Button key="receipt" variant="outline" onClick={()=>onReceipt({loan_id:a.id,loan_number:a.loan_number,reference:a.disbursement_transaction_id,amount:a.principal_amount,disbursed_at:a.disbursed_at,actor:"#"+a.created_by,branch:"KCP #"+a.disbursing_branch_id})}>{t("Lihat bukti pencairan")}</Button>],
        ]}/>}
 {(section === "summary" || section === "schedule") && <>
        <p>{t("Versi jadwal")} #{scheduleVersion} <Button variant="ghost" onClick={() => setVersion(a.current_schedule_version_id)}>{t("Versi saat ini")}</Button>
        </p>
        <Paged<Installment> key={scheduleVersion} path={root + "/installments?schedule_version_id=" + scheduleVersion} caption={t("Jadwal angsuran")}>{rows => <ScheduleTable rows={rows} actual/>}</Paged>
        </>}
 {section === "versions" && <Paged<Version> path={root + "/schedule-versions"} caption={t("Riwayat versi")}>{rows => <div className="space-y-4">{rows.map(v => <section key={v.id} className="rounded border p-4 space-y-2">
                <p>#{v.version_number} · {t(v.version_type === "initial" ? "Awal" : "Restrukturisasi")} · {v.is_current_version ? t("Versi saat ini") : t("Historis")}</p>
                <DetailList className="max-w-lg" rows={[
                    [t("Tanggal berlaku"), v.effective_date.slice(0, 10)],
                    [t("Bunga pinjaman"), v.interest_rate_bps / 100 + "% p.a."],
                    [t("Tenor pinjaman"), v.tenor_months + " " + t("bulan")],
                    [t("Alasan perubahan"), v.reason === "Initial approved schedule" ? t("Jadwal awal sesuai persetujuan") : v.reason],
                    [t("Disetujui oleh"), "#" + v.approved_by],
                    [t("Tanggal persetujuan"), formatLoanTimestamp(v.approved_at, locale)],
                ]}/>
                <Button variant="outline" onClick={() => { setVersion(v.id); setSection("schedule"); }}>{t("Lihat jadwal")}</Button>
                </section>)}</div>}</Paged>}
 {section === "payments" && <div className="space-y-5">{user.role === "teller" && a.status === "active" && <form onSubmit={review} className="space-y-3">
            <h4 className="font-semibold">{t("Catat pembayaran")}</h4>
            <Button type="submit" disabled={loading} variant={preview ? "outline" : "default"}>
              {t(loading ? "Memeriksa angsuran…" : "Cek angsuran")}
            </Button>
            </form>}
 <ErrorNotice error={error} onClose={()=>setError("")}/>{preview && <section className="space-y-3 rounded border p-4">
            <DetailList className="max-w-lg" rows={[
              [t("Nominal pembayaran (Rp)"), money(preview.amount)],
              ...(installment ? [
                [t("Angsuran / referensi jadwal"), installment.installment_number ?? "#" + installment.schedule_id] as const,
                [t("Pokok"), money(installment.principal_amount)] as const,
                [t("Bunga"), money(installment.interest_amount)] as const,
              ] : []),
              [t("Sisa pokok aktual"), money(preview.principal_before) + " → " + money(preview.principal_after)],
              [t("Sisa kewajiban terjadwal"), money(preview.outstanding_payable)],
            ]}/>
            {!installment && <Allocations rows={preview.allocations}/>}
            <div className="max-w-lg">
              <Field label={t("Catatan (opsional)")}>
                <Textarea rows={2} maxLength={500} value={note} onChange={e => setNote(e.target.value)}/>
              </Field>
            </div>
            <Button onClick={() => setConfirm(true)}>{t("Lanjutkan pembayaran")}</Button>
            <Modal open={confirm} onClose={() => setConfirm(false)} title={t("Konfirmasi pembayaran")}
              footer={<>
                <Button variant="outline" onClick={() => setConfirm(false)}>{t("Batal")}</Button>
                <Button onClick={() => {
                    setConfirm(false);
                    void execute(root + "/repayments", repaymentBody(preview, note));
                }}>{t("Konfirmasi pembayaran")}</Button>
              </>}>
              <DetailList rows={[
                [t("Anggota"), a.member_name],
                [t("Nomor pinjaman"), a.loan_number],
                [t("Nominal"), money(preview.amount)],
                [t("KCP pemroses"), user.branch_name],
                ...(note.trim() ? [[t("Catatan (opsional)"), note.trim()] as const] : []),
              ]}/>
            </Modal>
            </section>}

 <Paged<Payment> path={root + "/repayments"} caption={t("Riwayat pembayaran")}>{rows => <div className="space-y-4">{rows.map(r => <section key={r.id} className="rounded border p-4 space-y-2">
                <h4 className="font-semibold">{r.repayment_number} · {money(r.amount)}</h4>
                <DetailList className="max-w-lg" rows={[
                    [t("Tanggal pembayaran"), formatLoanTimestamp(r.paid_at, locale)],
                    [t("KCP pemroses"), r.branch],
                    [t("Petugas"), r.actor],
                    [t("Referensi"), r.transaction_id],
                    ...(r.note?.trim() ? [[t("Catatan (opsional)"), r.note] as const] : []),
                ]}/>
                <Allocations rows={r.allocations}/>
                <Button variant="outline" onClick={() => onReceipt({ loan_id: a.id, loan_number: a.loan_number, repayment_number: r.repayment_number, reference: r.transaction_id, amount: r.amount, paid_at: r.paid_at, actor: r.actor, branch: r.branch, allocations: r.allocations })}>{t("Lihat bukti pembayaran")}</Button>
                </section>)}</div>}</Paged>
        </div>}
 </article>;
}
