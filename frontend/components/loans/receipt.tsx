"use client";
import { DetailList } from "@/components/ui/detail-list";
import { formatLoanTimestamp, type Receipt, type Staff, type Account } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { Allocations, ErrorNotice, useRead } from "./shared";
export function LoanReceipt({ receipt, user, open }: {
    receipt: Receipt;
    user: Staff;
    open: (id: number) => void;
}) {
    const { t, money, locale } = usePreferences();
    const { data, error } = useRead<Account>(receipt.loan_id ? "/loan-accounts/" + receipt.loan_id : "");
    return <section className="loan-receipt rounded-lg border p-5 space-y-3">
        <ErrorNotice error={error ?? ""}/>
        <DetailList rows={[
            [t("Nomor pinjaman"), receipt.loan_number ?? data?.loan_number ?? receipt.label],
            ...(receipt.repayment_number ? [[t("Pembayaran"), receipt.repayment_number] as const] : []),
            ...(receipt.reference ? [[t("Referensi"), receipt.reference] as const] : []),
            ...(data ? [
                [t("Anggota"), data.member_name] as const,
                [t("Nomor rekening"), data.account_number] as const,
                [t("Pengajuan"), "PJ-" + data.application_id] as const,
            ] : []),
            ...(receipt.amount !== undefined ? [[t("Nominal"), money(receipt.amount)] as const] : []),
            ...(receipt.disbursed_at || receipt.paid_at ? [
                [t(receipt.disbursed_at ? "Tanggal pencairan" : "Tanggal pembayaran"), formatLoanTimestamp(receipt.disbursed_at ?? receipt.paid_at ?? null, locale)] as const,
                [t("Petugas"), receipt.actor ?? user.name] as const,
                [t("KCP pemroses"), receipt.branch ?? user.branch_name] as const,
            ] : []),
        ]}/>
 {receipt.allocations && <Allocations rows={receipt.allocations}/>}
 <div className="no-print flex gap-3">
    {receipt.reference && <Button variant="outline" disabled={!!receipt.loan_id && !data} onClick={() => window.print()}>{t("Cetak")}</Button>}{receipt.loan_id && <Button variant="outline" onClick={() => open(receipt.loan_id!)}>{t("Lihat rincian")}</Button>}</div>
 </section>;
}
