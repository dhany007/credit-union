"use client";
import { DetailList } from "@/components/ui/detail-list";
import { useState } from "react";
import { type Account, type Installment, type Version, type Receipt } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { Paged, ScheduleTable } from "./shared";
export function AccountDetail({ account: a, onReceipt }: {
    account: Account;
    onReceipt: (receipt: Receipt) => void;
}) {
    const { t, money, locale } = usePreferences();
    const [section, setSection] = useState<"summary" | "schedule" | "versions">("summary");
    const [version, setVersion] = useState(a.current_schedule_version_id);
    const root = "/loan-accounts/" + a.id;
    const scheduleVersion = section === "summary" ? a.current_schedule_version_id : version;
    const formatDate = (value: string) => new Date(value).toLocaleString(locale === "id" ? "id-ID" : "en-US", { timeZone: "Asia/Jakarta" });
    return <article className="space-y-5 rounded-lg border p-5">
    <h3 className="font-semibold">{a.loan_number} · {a.member_name}</h3>
    <p>{a.account_number} · {a.responsible_branch} · {t(a.status === "paid_off" ? "Lunas" : "Aktif")}</p>
    <nav className="flex flex-wrap gap-2" aria-label={t("Rincian pinjaman")}>{(["summary", "schedule", "versions"] as const).map((v, i) => <Button key={v} variant={section === v ? "default" : "outline"} aria-pressed={section === v} onClick={() => setSection(v)}>{[t("Ringkasan"), t("Jadwal angsuran"), t("Riwayat versi")][i]}</Button>)}</nav>
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
                <p>{v.effective_date.slice(0, 10)} · {v.interest_rate_bps / 100}% p.a. · {v.tenor_months} {t("bulan")}</p>
                <p>{v.reason} · {t("Disetujui oleh")} #{v.approved_by} · {formatDate(v.approved_at)} WIB</p>
                <Button variant="outline" onClick={() => { setVersion(v.id); setSection("schedule"); }}>{t("Lihat jadwal")}</Button>
                </section>)}</div>}</Paged>}
 </article>;
}
