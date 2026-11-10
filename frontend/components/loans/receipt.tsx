"use client";
import type { Receipt, Staff, Account } from "@/lib/loans";
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
    return <section role="status" className="loan-receipt rounded-lg border p-5 space-y-3">
 <p>{receipt.repayment_number ?? receipt.loan_number ?? receipt.label}</p>
 {receipt.reference && <p>{t("Referensi")}: {receipt.reference}</p>}
 {receipt.loan_id && <>
        <ErrorNotice error={error ?? ""}/>{data && <p>{data.member_name} · {data.account_number} · {data.loan_number} · PJ-{data.application_id}</p>}</>}
 {receipt.amount !== undefined && <p className="text-lg font-semibold">{money(receipt.amount)}</p>}
 {(receipt.disbursed_at || receipt.paid_at) && <p>{new Date((receipt.disbursed_at || receipt.paid_at)!).toLocaleString(locale === "id" ? "id-ID" : "en-US", { timeZone: "Asia/Jakarta" })} WIB · {receipt.actor ?? user.name} · {receipt.branch ?? user.branch_name}</p>}
 {receipt.allocations && <Allocations rows={receipt.allocations}/>}
 <div className="no-print flex gap-3">
    {receipt.reference && <Button variant="outline" disabled={!!receipt.loan_id && !data} onClick={() => window.print()}>{t("Cetak")}</Button>}{receipt.loan_id && <Button variant="outline" onClick={() => open(receipt.loan_id!)}>{t("Lihat rincian")}</Button>}</div>
 </section>;
}
