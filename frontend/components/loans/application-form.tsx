"use client";

import { useId, useRef, useState } from "react";
import { api } from "@/lib/api";
import { jakartaToday, type Branch, type Preview, type Staff } from "@/lib/loans";
import { usePreferences } from "@/components/preferences";
import { Button } from "@/components/ui/button";
import { AmountInput } from "@/components/amount-input";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { ErrorNotice, Field, SchedulePreview, selectClass, type Execute } from "./shared";

type Member = { id: number; name: string; account_number: string };

export function ApplicationForm({ user, branches, execute, onCancel, blocked }: {
  user: Staff; branches: Branch[]; execute: Execute; onCancel: () => void; blocked: boolean;
}) {
  const { t } = usePreferences();
  const formID = useId();
  const form = useRef<HTMLFormElement>(null);
  const lookupVersion = useRef(0);
  const previewVersion = useRef(0);
  const [account, setAccount] = useState("");
  const [amount, setAmount] = useState("");
  const [member, setMember] = useState<Member | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [estimating, setEstimating] = useState(false);
  const [estimate, setEstimate] = useState<Preview | null>(null);

  function invalidatePreview() { previewVersion.current++; setEstimate(null); }

  async function find(event: React.FormEvent) {
    event.preventDefault();
    const version = ++lookupVersion.current;
    setLoading(true); setError("");
    try {
      const result = await api<Member>("/members/by-account/" + account);
      if (version === lookupVersion.current) setMember(result);
    } catch (e) {
      if (version === lookupVersion.current) setError(e instanceof Error ? e.message : "Failed to fetch");
    } finally {
      if (version === lookupVersion.current) setLoading(false);
    }
  }

  async function preview() {
    if (!form.current) return;
    const data = new FormData(form.current);
    const version = ++previewVersion.current;
    setEstimating(true); setError(""); setEstimate(null);
    try {
      const result = await api<Preview>("/loans/preview", { method: "POST", body: JSON.stringify({
        amount: Number(data.get("amount")), tenor_months: Number(data.get("tenor")), expected_disbursement_date: jakartaToday(),
      }) });
      if (version === previewVersion.current) setEstimate(result);
    } catch (e) {
      if (version === previewVersion.current) setError(e instanceof Error ? e.message : "Failed to fetch");
    } finally { setEstimating(false); }
  }

  return <div className="flex min-h-0 flex-1 flex-col">
    <div className="min-h-0 overflow-y-auto px-5 py-4">
      <fieldset disabled={blocked} className="space-y-5">
        <legend className="sr-only">{t("Pengajuan pinjaman reguler")}</legend>
        <form onSubmit={find} className="flex flex-wrap items-end gap-3">
          <Field label={t("Nomor rekening")}>
            <Input autoFocus required pattern="[0-9]{10}" maxLength={10} value={account} onChange={e => {
              lookupVersion.current++; invalidatePreview(); setAccount(e.target.value); setMember(null); setLoading(false);
            }}/>
          </Field>
          <Button disabled={loading}>{t("Cari rekening")}</Button>
        </form>
        <ErrorNotice error={error} onClose={()=>setError("")}/>
        {member && <form id={formID} ref={form} className="grid gap-4 md:grid-cols-2"
          onChange={invalidatePreview} onSubmit={e => {
            e.preventDefault();
            const data = Object.fromEntries(new FormData(e.currentTarget));
            void execute("/loans", { member_id: member.id, amount: Number(data.amount),
              tenor_months: Number(data.tenor), responsible_branch_id: Number(data.branch), purpose: data.purpose, risk: data.risk });
          }}>
          <p className="md:col-span-2">{member.name} · {member.account_number}</p>
          <Field label={t("Nominal pengajuan (Rp)")}><AmountInput required name="amount" value={amount} onValueChange={setAmount}/></Field>
          <Field label={t("Tenor yang diajukan (bulan)")}><Input required name="tenor" type="number" min={1} max={120} step={1} defaultValue={12}/></Field>
          <Field label={t("KCP penanggung jawab")}><select name="branch" className={selectClass} defaultValue={user.branch_id}>{branches.map(b => <option key={b.id} value={b.id}>{b.name}</option>)}</select></Field>
          <Field label={t("Risiko awal untuk diverifikasi Supervisor")}><select name="risk" className={selectClass}><option value="low">{t("Rendah")}</option><option value="medium">{t("Sedang")}</option><option value="high">{t("Tinggi")}</option></select></Field>
          <Field label={t("Tujuan pinjaman")}><Textarea name="purpose" required minLength={5} maxLength={1000}/></Field>
        </form>}
        {estimate && <SchedulePreview key={JSON.stringify(estimate)} preview={estimate}/>}
      </fieldset>
    </div>
    <footer className="flex shrink-0 flex-wrap justify-end gap-3 border-t bg-background px-5 py-4">
      <Button type="button" variant="ghost" disabled={blocked} onClick={onCancel}>{t("Batal")}</Button>
      <Button type="button" variant="outline" disabled={blocked || !member || estimating} onClick={() => void preview()}>{t(estimating ? "Memuat..." : "Preview anuitas")}</Button>
      <Button type="submit" form={formID} disabled={blocked || !member}>{t("Kirim pengajuan")}</Button>
    </footer>
  </div>;
}
