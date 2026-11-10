"use client";
import { useEffect, useRef, useState } from "react";
import { submitLoanCommand } from "@/lib/loan-command";
import { LoanReceipt } from "./loans/receipt";
import { loanListReference } from "@/lib/loans";
import { api, ApiError } from "@/lib/api";
import type { Account, Application, Branch, Command, Receipt, Staff } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApplicationForm, ApplicationDetail } from "./loans/application";
import { AccountDetail } from "./loans/account";
import { LoanPolicies } from "./loans/policies";
import { ErrorNotice, Field, Paged, selectClass, useRead } from "./loans/shared";
type View = "applications" | "payouts" | "accounts";
export function LoanWorkspace({ user, branches, approvalLimit, refresh }: {
    user: Staff;
    branches: Branch[];
    approvalLimit: number;
    refresh: () => Promise<void>;
}) {
    const { t, money } = usePreferences();
    const [view, setView] = useState<View>("applications");
    const [filters, setFilters] = useState("");
    const [selected, setSelected] = useState<number | null>(null);
    const [intake, setIntake] = useState(false);
    const [policies, setPolicies] = useState(false);
    const [epoch, setEpoch] = useState(0);
    const [busy, setBusy] = useState(false);
    const [pending, setPending] = useState<Command | null>(null);
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
        setError("");
        setReceipt(null);
        try {
            setPending(command);
            const result = await submitLoanCommand(command, sessionStorage, storageKey, c => api<Receipt>(c.path, { method: "POST", headers: { "Idempotency-Key": c.key }, body: c.payload }));
            setPending(null);
            setReceipt({ ...result, loan_id: result.loan_id ?? (command.path.startsWith("/loan-accounts/") ? Number(command.path.split("/")[2]) : undefined), label: command.path === "/loan-interest/policies" ? t("Versi tarif") + " #" + result.id : result.id ? "PJ-" + result.id : undefined });
            setSelected(null);
            setIntake(false);
            setEpoch(v => v + 1);
            await refresh();
        }
        catch (e) {
            setError(e instanceof Error ? e.message : "Failed to fetch");
            if (e instanceof ApiError && e.status >= 400 && e.status < 500 && e.status !== 401 && e.status !== 403 && e.status !== 408 && e.status !== 429) {
                sessionStorage.removeItem(storageKey);
                setPending(null);
                if (e.status === 409) {
                    setSelected(null);
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
    function navigate(v: View) { setView(v); setSelected(null); setFilters(""); setIntake(false); setPolicies(false); }
    const listPath = (view === "accounts" ? "/loan-accounts" : "/loan-applications") + "?" + filters + (view === "payouts" ? "&payout_pending=true" : "");
    return <div className="space-y-5">
    <ErrorNotice error={error}/>{pending && <section role="status" className="rounded border border-primary p-4 space-y-3">
        <p>{t(busy ? "Menyimpan..." : "Hasil permintaan belum pasti. Jangan buat transaksi baru. Coba ulang dengan referensi yang sama.")}</p>
        <p className="break-all text-xs">{pending.key}</p>
        <Button disabled={busy} onClick={() => void submit(pending)}>{t("Coba ulang referensi yang sama")}</Button>
        </section>}{receipt && <LoanReceipt key={receipt.reference ?? receipt.label} receipt={receipt} user={user} open={id => { navigate("accounts"); setSelected(id); }}/>}

 <fieldset disabled={!ready || busy || !!pending} className="no-print space-y-5 min-w-0">
    <legend className="sr-only">{t("Pinjaman")}</legend>
    <nav className="flex flex-wrap gap-2" aria-label={t("Layanan pinjaman")}>{(["applications", "payouts", "accounts"] as const).map((v, i) => <Button key={v} aria-pressed={view === v && !policies} variant={view === v && !policies ? "default" : "outline"} onClick={() => navigate(v)}>{[t("Pengajuan"), t("Pencairan"), t("Daftar pinjaman")][i]}</Button>)}{user.role === "supervisor" && <Button variant={policies ? "default" : "outline"} onClick={() => { setPolicies(!policies); setSelected(null); }}>{t("Tarif bunga pinjaman")}</Button>}</nav>
 {policies ? <LoanPolicies key={epoch} execute={execute}/> : <>{user.role === "teller" && view === "applications" && <Button variant="outline" onClick={() => setIntake(!intake)}>{t("Ajukan pinjaman")}</Button>}{intake && <ApplicationForm user={user} branches={branches} execute={execute}/>}
 <form key={view} className="flex flex-wrap items-end gap-3" onSubmit={e => {
                e.preventDefault();
                const d = Object.fromEntries(new FormData(e.currentTarget));
                const q = new URLSearchParams();
                if (d.account)
                    q.set("account_number", String(d.account));
                if (d.reference)
                    q.set(view === "accounts" ? "loan_number" : "application_number", String(d.reference));
                q.set("status", view === "payouts" ? "approved" : String(d.status));
                setFilters(q.toString());
                setSelected(null);
            }}>
        <Field label={t("Nomor rekening")}>
        <Input name="account" pattern="[0-9]{10}" maxLength={10}/>
        </Field>
        <Field label={t("Referensi")}>
        <Input name="reference" maxLength={50} placeholder={view === "accounts" ? "LN-" : "PJ-"}/>
        </Field>{view !== "payouts" && <Field label={t("Status")}>
            <select name="status" className={selectClass} defaultValue={view === "accounts" ? "active" : "all"}>{(view === "accounts" ? ["active", "paid_off", "all"] : ["all", "submitted", "approved", "rejected"]).map(s => <option key={s} value={s}>{statusLabel(s, t)}</option>)}</select>
            </Field>}<Button>{t("Terapkan filter")}</Button>
        <Button type="button" variant="outline" onClick={() => { setEpoch(v => v + 1); setSelected(null); }}>{t("Muat ulang")}</Button>
        </form>
 <Paged<Application | Account> key={listPath + epoch} path={listPath} caption={t("Daftar pinjaman")}>{rows => <div className="overflow-x-auto">
            <table className="w-full whitespace-nowrap text-left text-sm">
            <caption className="sr-only">{t("Daftar pinjaman")}</caption>
            <thead>
            <tr>{[t("Referensi"), t("Anggota"), t("Nomor rekening"), t("Nominal"), t("Status"), t("Tindakan")].map(h => <th key={h} className="border-b p-3">{h}</th>)}</tr>
            </thead>
            <tbody>{rows.map(r => <tr key={r.id}>
                <td className="border-b p-3">{loanListReference(r)}</td>
                <td className="border-b p-3">{r.member_name}</td>
                <td className="border-b p-3">{r.account_number}</td>
                <td className="border-b p-3">{money("principal_amount" in r ? r.principal_amount : r.amount)}</td>
                <td className="border-b p-3">{statusLabel(r.status, t)}</td>
                <td className="border-b p-3">
                <Button variant="outline" onClick={() => setSelected(r.id)}>{t("Lihat rincian")}</Button>
                </td>
                </tr>)}</tbody>
            </table>
            </div>}</Paged>
 {selected && <Detail key={view + selected + epoch} id={selected} view={view} user={user} limit={approvalLimit} execute={execute} onReceipt={setReceipt}/>}</>}
 </fieldset>
    </div>;
}
function statusLabel(status: string, t: ReturnType<typeof usePreferences>["t"]) { return t(status === "active" ? "Aktif" : status === "paid_off" ? "Lunas" : status === "approved" ? "Disetujui" : status === "rejected" ? "Ditolak" : status === "submitted" ? "Menunggu keputusan" : "Semua"); }
function Detail({ id, view, user, limit, execute, onReceipt }: {
    id: number;
    view: View;
    user: Staff;
    limit: number;
    execute: (path: string, input: unknown) => Promise<void>;
    onReceipt: (receipt: Receipt) => void;
}) {
    const { t } = usePreferences();
    const { data, error } = useRead<Application | Account>((view === "accounts" ? "/loan-accounts/" : "/loan-applications/") + id);
    return <>
    <ErrorNotice error={error ?? ""}/>{data ? view === "accounts" ? <AccountDetail account={data as Account} user={user} execute={execute} onReceipt={onReceipt}/> : <ApplicationDetail application={data as Application} user={user} limit={limit} payout={view === "payouts"} execute={execute}/> : !error && <p role="status">{t("Memuat...")}</p>}</>;
}
