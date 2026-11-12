"use client";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { loanRoutes, newApplicationRoute } from "@/lib/workspace-routes";
import { toast } from "sonner";
import { ArrowLeft } from "lucide-react";
import { submitLoanCommand } from "@/lib/loan-command";
import { LoanReceipt } from "./loans/receipt";
import { applicationStage, formatLoanTimestamp, loanListReference } from "@/lib/loans";
import { api, ApiError } from "@/lib/api";
import type { Account, Application, Branch, Command, Receipt, Staff } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApplicationForm } from "./loans/application-form";
import { Modal } from "./modal";
import { localizedError } from "@/lib/api-errors";
import { ApplicationDetail } from "./loans/application";
import { AccountDetail } from "./loans/account";
import { LoanPolicies } from "./loans/policies";
import { ErrorNotice, Field, Paged, selectClass, useRead } from "./loans/shared";
type View = "applications" | "accounts";
export function LoanWorkspace({ user, branches, refresh, section, selected, intake }: {
    section: View | "policies";
    selected: number | null;
    intake: boolean;
    user: Staff;
    branches: Branch[];
    refresh: () => Promise<void>;
}) {
    const { t, money, locale } = usePreferences();
    const router = useRouter();
    const view = section === "accounts" ? "accounts" : "applications";
    const listHref = loanRoutes[view];
    const [filters, setFilters] = useState("");
    const policies = section === "policies" && user.role === "supervisor";
    const [epoch, setEpoch] = useState(0);
    const [busy, setBusy] = useState(false);
    const [pending, setPending] = useState<Command | null>(null);
    const [pendingHidden, setPendingHidden] = useState(false);
    const [ready, setReady] = useState(false);
    const [error, setError] = useState("");
    const [receipt, setReceipt] = useState<Receipt | null>(null);
    const lock = useRef(false);
    const storageKey = "credit-union.loan-command." + user.id;
    // Hydrate the saved cash command before enabling any new writes after SSR.
    useEffect(() => {
        try {
            const raw = sessionStorage.getItem(storageKey);
            if (raw) {
                const c: Command = JSON.parse(raw);
                if (typeof c.path !== "string" || typeof c.key !== "string" || typeof c.payload !== "string" || !/^\/(loans|loan-applications|loan-accounts|loan-interest)(\/|$)/.test(c.path))
                    throw Error("Invalid request reference");
                // eslint-disable-next-line react-hooks/set-state-in-effect -- Restore browser-only storage before enabling cash commands after SSR.
                setPending(c);
            }
            setReady(true);
        }
        catch {
            setError("Request reference storage unavailable");
        }
    }, [storageKey]);
    async function submit(command: Command) {
        if (lock.current)
            return;
        lock.current = true;
        setBusy(true);
        setPendingHidden(false);
        setError("");
        setReceipt(null);
        try {
            setPending(command);
            const result = await submitLoanCommand(command, sessionStorage, storageKey, c => api<Receipt>(c.path, { method: "POST", headers: { "Idempotency-Key": c.key }, body: c.payload }));
            setPending(null);
            if (command.path.endsWith("/disburse")) {
                const loanID = result.loan_id;
                toast.success(t("Pinjaman berhasil dicairkan."), {
                    description: result.loan_number ?? result.reference,
                    action: loanID ? {
                        label: t("Lihat rincian"),
                        onClick: () => router.push(loanRoutes.accounts + "/" + loanID),
                    } : undefined,
                });
            } else if (result.reference || result.loan_id) {
                setReceipt({ ...result, loan_id: result.loan_id ?? (command.path.startsWith("/loan-accounts/") ? Number(command.path.split("/")[2]) : undefined), label: result.id ? "PJ-" + result.id : undefined });
            } else {
                const decision = command.path.endsWith("/decision") ? JSON.parse(command.payload).decision : null;
                toast.success(t(command.path === "/loans" ? "Pengajuan pinjaman berhasil dikirim." : decision === "approved" ? "Pengajuan pinjaman disetujui." : decision === "rejected" ? "Pengajuan pinjaman ditolak." : "Permintaan berhasil dicatat"), {
                    description: command.path === "/loan-interest/policies" ? t("Versi tarif") + " #" + result.id : result.id ? "PJ-" + result.id : undefined,
                });
            }
            if (command.path === "/loans" && window.location.pathname === newApplicationRoute) router.replace(listHref);
            setEpoch(v => v + 1);
            await refresh();
        }
        catch (e) {
            setError(e instanceof Error ? e.message : "Failed to fetch");
            if (e instanceof ApiError && e.status >= 400 && e.status < 500 && e.status !== 401 && e.status !== 403 && e.status !== 408 && e.status !== 429) {
                setPending(null);
                if (e.status === 409 || (e.status === 422 && command.path.endsWith("/disburse"))) {
                    setEpoch(v => v + 1);
                }
            }
        }
        finally {
            lock.current = false;
            setBusy(false);
        }
    }
    async function execute(path: string, input: unknown) {
        if (pending || lock.current)
            return;
        await submit({ path, payload: JSON.stringify(input), key: crypto.randomUUID() });
    }
    const listPath = (view === "accounts" ? "/loan-accounts" : "/loan-applications") + "?" + filters;
    const dateColumn = {
        label: t(view === "accounts" ? "Tanggal pencairan" : "Tanggal pengajuan"),
        render: (row: Application | Account) => formatLoanTimestamp(
            "application_number" in row ? row.created_at : row.disbursed_at,
            locale,
        ),
    };
    const columns: { label: string; render: (row: Application | Account) => ReactNode }[] = [
        { label: t("Referensi"), render: loanListReference },
        dateColumn,
        { label: t("Anggota"), render: row => row.member_name },
        { label: t("Nomor rekening"), render: row => row.account_number },
        { label: t("Nominal"), render: row => money("application_number" in row ? row.amount : row.principal_amount) },
        {
            label: t("Status"),
            render: row => "application_number" in row ? t(applicationStage(row)) : statusLabel(row.status, t),
        },
    ];
    if (view === "applications") {
        columns.push({
            label: t("Tanggal keputusan"),
            render: row => formatLoanTimestamp("application_number" in row ? row.decided_at : null, locale),
        });
    }
    columns.push({
        label: t("Tindakan"),
        render: row => <Button variant="outline" onClick={() => router.push(listHref + "/" + row.id)}>{t("Lihat rincian")}</Button>,
    });
    if ((intake && user.role !== "teller") || (section === "policies" && !policies)) {
        return <p role="status">{t("Anda tidak memiliki akses ke halaman ini.")}</p>;
    }
    return <div className="space-y-5">
    {!pending && <ErrorNotice error={error} onClose={()=>setError("")}/>}
    {pending && pendingHidden && <Button variant="outline" onClick={()=>setPendingHidden(false)}>{t("Pulihkan permintaan")}</Button>}
    <Modal open={!!pending && !pendingHidden} onClose={()=>setPendingHidden(true)}
      title={t(busy ? "Menyimpan..." : "Pemulihan transaksi")} blocked={busy}
      footer={<Button disabled={busy} onClick={()=>{if(pending) void submit(pending);}}>{t("Coba ulang referensi yang sama")}</Button>}>
      <p className="text-sm">{t(busy ? "Mohon tunggu hingga proses selesai." : "Hasil permintaan belum pasti. Jangan buat transaksi baru. Coba ulang dengan referensi yang sama.")}</p>
      {error && <p role="alert" className="mt-3 text-sm text-destructive">{localizedError(locale,error)}</p>}
      <p className="mt-3 break-all font-mono text-xs text-muted-foreground">{pending?.key}</p>
    </Modal>
    <Modal open={!!receipt} onClose={()=>setReceipt(null)} title={t(receipt?.disbursed_at ? "Bukti pencairan" : "Permintaan berhasil dicatat")} wide={!!receipt?.loan_id}>
      {receipt && <LoanReceipt key={receipt.reference ?? receipt.label} receipt={receipt} user={user}
        open={id=>{setReceipt(null);router.push(loanRoutes.accounts + "/" + id);}}/>}
    </Modal>
    {intake && <section className="space-y-4">
      <Button variant="ghost" disabled={busy} onClick={() => router.replace(listHref)}><ArrowLeft aria-hidden="true" size={18}/>{t("Kembali ke daftar pengajuan")}</Button>
      <h2 className="text-xl font-semibold">{t("Pengajuan pinjaman reguler")}</h2>
      <ApplicationForm user={user} branches={branches} execute={execute} onCancel={()=>router.replace(listHref)} blocked={!ready || busy || !!pending}/>
    </section>}
    {selected !== null && <section aria-label={t("Rincian pinjaman")} className="space-y-5">
      <header className="space-y-3">
        <Button variant="ghost" disabled={busy} onClick={() => { router.replace(listHref); }}>
          <ArrowLeft aria-hidden="true" size={18}/>{t(view === "accounts" ? "Kembali ke daftar pinjaman" : "Kembali ke daftar pengajuan")}
        </Button>
        <h2 tabIndex={-1} ref={node => node?.focus()} className="text-xl font-semibold">{t("Rincian pinjaman")}</h2>
      </header>
      <fieldset disabled={!ready || busy || !!pending} className="min-w-0">
        <Detail key={view + selected + epoch} id={selected} view={view} user={user} execute={execute} onReceipt={setReceipt}/>
      </fieldset>
    </section>}
    <div hidden={selected !== null || intake}>

 <fieldset disabled={!ready || busy || !!pending} className="no-print space-y-5 min-w-0">
    <legend className="sr-only">{t("Pinjaman")}</legend>

 {policies ? <LoanPolicies key={epoch} execute={execute}/> : <>{user.role === "teller" && view === "applications" && <Button variant="outline" onClick={() => router.push(newApplicationRoute)}>{t("Ajukan pinjaman")}</Button>}
 <form key={view} className="flex flex-wrap items-end gap-3" onSubmit={e => {
                e.preventDefault();
                const d = Object.fromEntries(new FormData(e.currentTarget));
                const q = new URLSearchParams();
                if (d.account)
                    q.set("account_number", String(d.account));
                if (d.reference)
                    q.set(view === "accounts" ? "loan_number" : "application_number", String(d.reference));
                q.set("status", String(d.status));
                if (view === "applications" && d.status === "approved") q.set("payout_pending", "true");
                setFilters(q.toString());
            }}>
        <Field label={t("Nomor rekening")}>
        <Input name="account" pattern="[0-9]{10}" maxLength={10}/>
        </Field>
        <Field label={t("Referensi")}>
        <Input name="reference" maxLength={50} placeholder={view === "accounts" ? "LN-" : "PJ-"}/>
        </Field>{<Field label={t("Status")}>
            <select name="status" className={selectClass} defaultValue={view === "accounts" ? "active" : "all"}>{(view === "accounts" ? ["active", "paid_off", "all"] : ["all", "submitted", "approved", "rejected"]).map(s => <option key={s} value={s}>{statusLabel(s, t)}</option>)}</select>
            </Field>}<Button>{t("Terapkan filter")}</Button>
        <Button type="button" variant="outline" onClick={() => { setEpoch(v => v + 1); }}>{t("Muat ulang")}</Button>
        </form>
 <Paged<Application | Account> key={listPath + epoch} path={listPath} caption={t("Daftar pinjaman")}>{rows => <div className="overflow-x-auto">
            <table className="w-full whitespace-nowrap text-left text-sm">
            <caption className="sr-only">{t("Daftar pinjaman")}</caption>
            <thead>
                <tr>{columns.map(column => <th key={column.label} scope="col" className="border-b p-3">{column.label}</th>)}</tr>
            </thead>
            <tbody>
                {rows.map(row => <tr key={row.id}>
                    {columns.map(column => <td key={column.label} className="border-b p-3">{column.render(row)}</td>)}
                </tr>)}
            </tbody>
            </table>
            </div>}</Paged>
 </>}
 </fieldset>
    </div>
    </div>;
}
function statusLabel(status: string, t: ReturnType<typeof usePreferences>["t"]) { return t(status === "active" ? "Aktif" : status === "paid_off" ? "Lunas" : status === "approved" ? "Menunggu pencairan" : status === "rejected" ? "Ditolak" : status === "submitted" ? "Menunggu persetujuan" : "Semua"); }
function Detail({ id, view, user, execute, onReceipt }: {
    id: number;
    view: View;
    user: Staff;
    execute: (path: string, input: unknown) => Promise<void>;
    onReceipt: (receipt: Receipt) => void;
}) {
    const { t } = usePreferences();
    const { data, error } = useRead<Application | Account>((view === "accounts" ? "/loan-accounts/" : "/loan-applications/") + id);
    const authority = useRead<{ approval_limit: number }>(view === "accounts" ? "" : "/meta");
    return <>
    <ErrorNotice error={error ?? authority.error ?? ""}/>{data ? view === "accounts" ? <AccountDetail account={data as Account} onReceipt={onReceipt}/> : authority.data ? <ApplicationDetail application={data as Application} user={user} limit={authority.data.approval_limit} payout={user.role === "supervisor" && (data as Application).status === "approved" && !(data as Application).linked_loan_id && (data as Application).pricing_complete} execute={execute}/> : !authority.error && <p role="status">{t("Memuat...")}</p> : !error && <p role="status">{t("Memuat...")}</p>}</>;
}
