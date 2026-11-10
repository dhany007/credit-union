"use client";
import { LoanWorkspace } from "@/components/loan-workspace";

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import {
  ArrowDownLeft,
  ArrowUpRight,
  Building2,
  Check,
  ChevronLeft,
  ChevronRight,
  CreditCard,
  Landmark,
  Loader2,
  LogOut,
  Plus,
  Search,
  ShieldCheck,
  Users,
  Wallet,
  ReceiptText,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { AmountInput } from "@/components/amount-input";
import { Input } from "@/components/ui/input";
import { PreferenceControls, usePreferences } from "@/components/preferences";
import { type Notice } from "@/lib/messages";
import { ErrorNotice, Notification, Modal } from "@/components/modal";
import { Textarea } from "@/components/ui/textarea";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { api } from "@/lib/api";
import { InterestSettings } from "@/components/interest-settings";
import { movementLabel, movementSign } from "@/lib/interest";
import { TransactionList } from "@/components/transaction-list";

type User = {
  id: number;
  name: string;
  role: "teller" | "supervisor";
  branch_id: number;
  branch_name: string;
};
type Member = {
  id: number;
  number: string;
  account_number: string;
  name: string;
  nik_masked: string;
  phone: string;
  address: string;
  home_branch: string;
  balance: number;
};
type Meta = {
  branches: { id: number; name: string }[];
  branch_cash: number;
  approval_limit: number;
};
type Transaction = {
  balance_before: number | null;
  balance_after: number | null;
  reference: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
  branch: string;
  actor: string;
};
function loadWorkspace(searchQuery: string, signal?: AbortSignal) {
  return Promise.all([
    api<Member[]>(`/members?search=${encodeURIComponent(searchQuery)}`, { signal }),
    api<Meta>("/meta", { signal }),
  ]);
}

function Field({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <label className="grid gap-2 text-sm">
      <span className="font-medium">{label}</span>
      {children}
    </label>
  );
}
const TABLE_PAGE_SIZE = 20;
export default function Home() {
  const { t, money, ready } = usePreferences();
  const [user, setUser] = useState<User | null>(null),
    [boot, setBoot] = useState(true);
  const [tab, setTab] = useState("members"),
    [members, setMembers] = useState<Member[]>([]),
    [meta, setMeta] = useState<Meta | null>(null);
  const [selected, setSelected] = useState<number | null>(null),
    [historyResult, setHistoryResult] = useState<{
      memberID: number | null;
      actorID: number;
      rows: Transaction[];
    } | null>(null);
  const history = historyResult?.memberID === selected && historyResult?.actorID === user?.id
    ? historyResult.rows
    : [];
  const [search, setSearch] = useState(""),
    [query, setQuery] = useState(""),
    [memberPage, setMemberPage] = useState(1),
    [newMember, setNewMember] = useState(false);
  const [error, setError] = useState(""),
    [success, setSuccess] = useState<Notice | null>(null),
    [busy, setBusy] = useState(false);
  const isCash = tab === "deposit" || tab === "withdrawal";
  const cashKind = tab === "withdrawal" ? "withdrawal" : "deposit";
  const memberPageCount = Math.max(1, Math.ceil(members.length / TABLE_PAGE_SIZE));
  const currentMemberPage = Math.min(memberPage, memberPageCount);
  const visibleMembers = members.slice(
    (currentMemberPage - 1) * TABLE_PAGE_SIZE,
    currentMemberPage * TABLE_PAGE_SIZE,
  );
  const memberRowStart = members.length ? (currentMemberPage - 1) * TABLE_PAGE_SIZE + 1 : 0;
  const memberRowEnd = Math.min(currentMemberPage * TABLE_PAGE_SIZE, members.length);
  const [cashAmount, setCashAmount] = useState(""),
    [cashNote, setCashNote] = useState(""),
    [confirmCash, setConfirmCash] = useState(false);
  const [lastReceipt, setLastReceipt] = useState<string | null>(null);
  const requestRef = useRef<Record<string, { payload: string; key: string }>>(
    {},
  );
  const [resolvedMember, setResolvedMember] = useState<Member | null>(null);
  const [accountInput, setAccountInput] = useState("");
  const accountLookupVersion = useRef(0);
  const member =
    members.find((m) => m.id === selected) ??
    (resolvedMember?.id === selected ? resolvedMember : undefined);
  async function refresh(searchQuery = query) {
    const [ms, mt] = await loadWorkspace(searchQuery);
    setMembers(ms);
    setMeta(mt);
    return ms;
  }
  useEffect(() => {
    api<User>("/me")
      .then(setUser)
      .catch(() => {})
      .finally(() => setBoot(false));
  }, []);
  useEffect(() => {
    if (!user) return;
    const controller = new AbortController();
    loadWorkspace(query, controller.signal)
      .then(([ms, mt]) => {
        if (controller.signal.aborted) return;
        setMembers(ms);
        setMeta(mt);
      })
      .catch((error) => {
        if (!controller.signal.aborted)
          setError(error instanceof Error ? error.message : "Unexpected error");
      });
    return () => controller.abort();
  }, [user, query]);
  useEffect(() => {
    if (!user || !selected) return;
    const controller = new AbortController();
    api<Transaction[]>(`/members/${selected}/transactions`, { signal: controller.signal })
      .then((rows) => {
        if (!controller.signal.aborted)
          setHistoryResult({ memberID: selected, actorID: user.id, rows });
      })
      .catch((error) => {
        if (!controller.signal.aborted)
          setError(error instanceof Error ? error.message : "Unexpected error");
      });
    return () => controller.abort();
  }, [user, selected]);
  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError("");
    setSuccess(null);
    try {
      await fn();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unexpected error");
    } finally {
      setBusy(false);
    }
  }
  async function mutate<T>(path: string, input: unknown): Promise<T> {
    const payload = JSON.stringify(input);
    let entry = requestRef.current[path];
    if (!entry || entry.payload !== payload) {
      entry = { payload, key: crypto.randomUUID() };
      requestRef.current[path] = entry;
    }
    const result = await api<T>(path, {
      method: "POST",
      headers: { "Idempotency-Key": entry.key },
      body: payload,
    });
    delete requestRef.current[path];
    return result;
  }
  function navigate(value: string) {
    accountLookupVersion.current++;
    if (value === "deposit" || value === "withdrawal") {
      setAccountInput(member?.account_number ?? "");
      setCashAmount("");
      setCashNote("");
    }
    setTab(value);
    setError("");
    setSuccess(null);
    setConfirmCash(false);
  }
  async function lookupAccount() {
    if (busy) return;
    const version = ++accountLookupVersion.current;
    const number = accountInput.trim();
    setSelected(null);
    setResolvedMember(null);
    setConfirmCash(false);
    await run(async () => {
      const found = await api<Member>(
        `/members/by-account/${encodeURIComponent(number)}`,
      );
      if (version !== accountLookupVersion.current) return;
      setMembers((current) => [
        found,
        ...current.filter((item) => item.id !== found.id),
      ]);
      setResolvedMember(found);
      setSelected(found.id);
      setAccountInput(found.account_number);
    });
  }
  if (boot || !ready)
    return (
      <main className="grid min-h-screen place-items-center">
        <Loader2 className="animate-spin" aria-label={t("Memuat")} />
      </main>
    );
  if (!user)
    return (
      <main className="grid min-h-screen lg:grid-cols-2">
        <section className="hidden flex-col justify-between bg-primary p-14 text-primary-foreground lg:flex">
          <div className="flex items-center gap-3 text-lg">
            <Landmark /> Credit Union
          </div>
          <div>
            <p className="mb-6 text-xs uppercase tracking-[.25em] opacity-60">
              {t("Tumbuh bersama anggota")}
            </p>
            <h1 className="max-w-lg font-serif text-6xl leading-tight">
              {t("Pelayanan yang dekat.")}
              <br />
              {t("Keuangan yang tercatat.")}
            </h1>
            <p className="mt-6 max-w-sm text-sm leading-7 opacity-70">
              {t(
                "Satu keanggotaan, layanan di setiap KCP. Kelola kebutuhan anggota dalam satu ruang kerja.",
              )}
            </p>
          </div>
          <span className="text-xs opacity-50">
            {t("Ruang kerja operasional koperasi")}
          </span>
        </section>
        <section className="flex items-center justify-center p-6">
          <div className="w-full max-w-sm">
            <div className="mb-8 flex justify-end">
              <PreferenceControls />
            </div>
            <Landmark className="mb-10 h-9 w-9 lg:hidden" />
            <Badge variant="outline" className="mb-5">
              {t("AKSES PETUGAS")}
            </Badge>
            <h2 className="font-serif text-4xl">{t("Selamat datang.")}</h2>
            <p className="mb-8 mt-3 text-sm text-muted-foreground">
              {t("Masuk untuk melayani anggota di KCP Anda.")}
            </p>
            <form
              className="grid gap-5"
              onSubmit={(e) => {
                e.preventDefault();
                const data = new FormData(e.currentTarget);
                run(async () => {
                  await api("/login", {
                    method: "POST",
                    body: JSON.stringify(Object.fromEntries(data)),
                  });
                  setUser(await api<User>("/me"));
                });
              }}
            >
              <Field label={t("Username")}>
                <Input
                  name="username"
                  autoComplete="username"
                  required
                  placeholder={t("Username petugas")}
                />
              </Field>
              <Field label={t("Kata sandi")}>
                <Input
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                  placeholder={t("Masukkan kata sandi")}
                />
              </Field>
              <ErrorNotice error={error} onClose={()=>setError("")}/>
              <Button disabled={busy} className="mt-2 h-11">
                {busy ? (
                  <Loader2 className="animate-spin" />
                ) : (
                  <>
                    {t("Masuk ke ruang kerja")}{" "}
                    <ArrowUpRight className="ml-auto" />
                  </>
                )}
              </Button>
            </form>
            <p className="mt-8 text-xs leading-5 text-muted-foreground">
              {t(
                "Akses sesuai peran Teller atau Supervisor. Hubungi pengelola jika Anda belum memiliki akun.",
              )}
            </p>
          </div>
        </section>
      </main>
    );
  return (
    <div className="min-h-screen lg:grid lg:grid-cols-[240px_1fr]">
      <aside className="no-print flex flex-col border-b bg-[var(--sidebar)] p-5 lg:sticky lg:top-0 lg:h-screen lg:border-b-0 lg:border-r lg:p-6">
        <Link href="/" className="mb-8 flex items-center gap-3 font-semibold">
          <span className="grid h-10 w-10 place-items-center rounded-lg bg-primary text-primary-foreground">
            <Landmark size={20} />
          </span>
          <span>
            Credit Union
            <span className="mt-1 block text-[10px] font-normal uppercase tracking-[.2em] text-muted-foreground">
              {t("Ruang kerja KCP")}
            </span>
          </span>
        </Link>
        <p className="mb-3 hidden text-[10px] uppercase tracking-[.2em] text-muted-foreground lg:block">
          {t("Layanan utama")}
        </p>
        <nav
          aria-label={t("Layanan utama")}
          className="flex flex-wrap gap-1 lg:flex-col"
        >
          {[
            { id: "members", label: t("Anggota"), icon: Users },
            { id: "deposit", label: t("Setor Tunai"), icon: ArrowDownLeft },
            { id: "withdrawal", label: t("Tarik Tunai"), icon: ArrowUpRight },
            { id: "loans", label: t("Pinjaman"), icon: CreditCard },
            { id: "transactions", label: t("Transaksi"), icon: ReceiptText },
 ...(user.role === "supervisor" ? [{ id: "interest", label: t("Bunga simpanan"), icon: Wallet }] : []),
          ].map((item) => (
            <button
              key={item.id}
              onClick={() => navigate(item.id)}
              disabled={busy}
              aria-current={tab === item.id ? "page" : undefined}
              className={cn(
                "flex flex-1 items-center gap-3 rounded-md px-3 py-3 text-sm transition-colors lg:flex-none",
                tab === item.id
                  ? "bg-primary text-primary-foreground"
                  : "hover:bg-accent",
              )}
            >
              <item.icon size={18} />
              {item.label}
              {tab === item.id && (
                <ChevronRight className="ml-auto hidden lg:block" size={14} />
              )}
            </button>
          ))}
        </nav>
        <div className="mt-auto hidden border-t pt-5 lg:block">
          <p className="mb-2 flex items-center gap-2 text-xs font-medium">
            <Building2 size={14} />
            {user.branch_name}
          </p>
          <p className="text-xs leading-5 text-muted-foreground">
            {t("Keanggotaan terpusat.")}
            <br />
            {t("Layanan lintas KCP.")}
          </p>
        </div>
      </aside>
      <div className="min-w-0">
        <header className="no-print flex flex-wrap items-center justify-between gap-3 border-b px-5 py-4 lg:px-10">
          <p className="flex items-center gap-2 text-xs text-muted-foreground">
            <span className="hidden sm:inline">{t("Operasional")}</span>
            <ChevronRight size={12} />
            <span className="text-foreground">{user.branch_name}</span>
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <PreferenceControls />
            <div className="text-right">
              <p className="text-sm font-medium">{user.name}</p>
              <p className="text-[11px] capitalize text-muted-foreground">
                {user.role}
              </p>
            </div>
            <Button
              variant="ghost"
              size="icon"
              title={t("Keluar")}
              aria-label={t("Keluar")}
              onClick={() =>
                run(async () => {
                  await api("/logout", { method: "POST" });
                  setUser(null);
 setTab("members");
                  setResolvedMember(null);
                  setAccountInput("");
                  accountLookupVersion.current++;
                  setMembers([]);
                  setSelected(null);
                  setHistoryResult(null);
                  setMeta(null);
                  setError("");
                  setSuccess(null);
                })
              }
            >
              <LogOut size={17} />
            </Button>
          </div>
        </header>
        <main className="mx-auto max-w-7xl px-5 py-8 lg:px-10 lg:py-10">
          <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="mb-2 text-[10px] uppercase tracking-[.24em] text-muted-foreground">
                {tab === "members"
                  ? t("Kenali anggota Anda")
                  : isCash
                    ? t("Transaksi tunai")
                    : tab === "transactions"
                      ? t("Riwayat simpanan")
 : tab === "interest" ? t("Pengaturan tarif")
                      : t("Pengajuan & keputusan")}
              </p>
              <h1 className="font-serif text-4xl">
                {tab === "members"
                  ? t("Keanggotaan")
                  : isCash
                    ? t(cashKind === "deposit" ? "Setor Tunai" : "Tarik Tunai")
                    : tab === "transactions"
                      ? t("Transaksi")
 : tab === "interest" ? t("Bunga simpanan")
                      : t("Layanan pinjaman")}
              </h1>
              <p className="mt-3 max-w-xl text-sm leading-6 text-muted-foreground">
                {tab === "members"
                  ? t("Temukan dan layani anggota dari seluruh KCP.")
                  : isCash
                    ? t(
                        "Catat transaksi pada rekening anggota yang sama, dari KCP mana saja.",
                      )
                    : tab === "transactions"
                      ? t(
                          "Tinjau setoran, penarikan, dan bunga anggota dari seluruh KCP.",
                        )
                      : tab === "interest" ? t("Kelola tarif tahunan dan pantau kredit bunga bulanan.")
 : t(
                          "Kelola pengajuan, pencairan, dan pembayaran pinjaman.",
                        )}
              </p>
            </div>
            {user.role === "teller" &&
              tab === "members" && (
                <Button
                  onClick={() => {
                    if (tab === "members") setNewMember(!newMember);
                  }}
                >
                  <Plus size={16} />
                  {tab === "members"
                    ? t("Daftarkan anggota")
                    : t("Ajukan pinjaman")}
                </Button>
              )}
          </div>
          <ErrorNotice error={error} onClose={()=>setError("")}/>
          {success && (
            <Notification title={t("Permintaan berhasil dicatat")} onClose={()=>setSuccess(null)}>
              <Check size={18} />
              <span>
                {t(success.key, {
                  ...success.values,
                  ...(success.kind
                    ? {
                        kind: t(
                          success.kind === "deposit" ? "Setoran" : "Penarikan",
                        ),
                      }
                    : {}),
                })}
              </span>
              {lastReceipt && (
                <Button
                  className="ml-auto"
                  variant="outline"
                  size="sm"
                  onClick={() => window.print()}
                >
                  {t("Cetak bukti")}
                </Button>
              )}
            </Notification>
          )}
          {tab === "members" && (
            <>
              {newMember && user.role === "teller" && (
                <Card className="mb-6">
                  <CardContent className="pt-6">
                    <h2 className="mb-2 text-lg font-semibold">
                      {t("Pendaftaran anggota")}
                    </h2>
                    <p className="mb-6 text-sm text-muted-foreground">
                      {t(
                        "Periksa KTP asli. Setoran awal {amount} akan dicatat sebagai saldo simpanan di {branch}.",
                        { amount: money(50000), branch: user.branch_name },
                      )}
                    </p>
                    <form
                      className="grid gap-5 md:grid-cols-2"
                      onSubmit={(e) => {
                        e.preventDefault();
                        const form = e.currentTarget;
                        const input = Object.fromEntries(new FormData(form));
                        run(async () => {
                          const result = await mutate<{
                            id: number;
                            number: string;
                            account_number?: string;
                            reference: string;
                          }>("/members", input);
                          setQuery("");
                          setSearch("");
                          setMemberPage(1);
                          const currentMembers = await refresh("");
                          const current =
                            currentMembers.find(
                              (item) => item.id === result.id,
                            ) ??
                            (
                              await api<Member[]>(
                                `/members?search=${encodeURIComponent(result.number)}`,
                              )
                            ).find((item) => item.id === result.id);
                          const accountNumber =
                            result.account_number ?? current?.account_number;
                          if (!accountNumber)
                            throw new Error("Savings account not found");
                          setResolvedMember(current ?? null);
                          setSelected(result.id);
                          form.reset();
                          setNewMember(false);
                          setLastReceipt(result.reference);
                          setSuccess({
                            key: "Anggota {number} terdaftar. Rekening {account}. Setoran awal Rp50.000 diterima. Referensi: {reference}",
                            values: {
                              number: result.number,
                              account: accountNumber,
                              reference: result.reference,
                            },
                          });
                        });
                      }}
                    >
                      <Field label={t("Nama sesuai KTP")}>
                        <Input
                          name="name"
                          minLength={2}
                          maxLength={120}
                          required
                        />
                      </Field>
                      <Field label={t("NIK")}>
                        <Input
                          name="nik"
                          inputMode="numeric"
                          pattern="[0-9]{16}"
                          minLength={16}
                          maxLength={16}
                          required
                          placeholder={t("16 digit NIK")}
                        />
                      </Field>
                      <Field label={t("Nomor telepon")}>
                        <Input
                          name="phone"
                          type="tel"
                          minLength={6}
                          maxLength={30}
                          required
                        />
                      </Field>
                      <Field label={t("Alamat")}>
                        <Input
                          name="address"
                          minLength={5}
                          maxLength={500}
                          required
                        />
                      </Field>
                      <div className="flex items-center gap-3 md:col-span-2">
                        <Button disabled={busy}>
                          {busy ? <Loader2 className="animate-spin" /> : null}
                          {t("Daftar")}
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          onClick={() => setNewMember(false)}
                        >
                          {t("Batal")}
                        </Button>
                      </div>
                    </form>
                  </CardContent>
                </Card>
              )}
              <div className="grid items-start gap-6 xl:grid-cols-[1fr_300px]">
                <Card className="gap-0 overflow-hidden py-0">
                  <div className="flex flex-wrap items-center justify-between gap-3 border-b p-5">
                    <div>
                      <h2 className="text-sm font-semibold">
                        {t("Daftar anggota")}
                      </h2>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {t("{count} anggota ditampilkan · semua KCP", {
                          count: members.length,
                        })}
                      </p>
                    </div>
                    <form
                      className="relative"
                      onSubmit={(e) => {
                        e.preventDefault();
                        setQuery(search);
                        setSelected(null);
                        setMemberPage(1);
                      }}
                    >
                      <Search
                        className="absolute left-3 top-3 text-muted-foreground"
                        size={15}
                      />
                      <Input
                        aria-label={t("Cari anggota")}
                        value={search}
                        onChange={(e) => setSearch(e.target.value)}
                        className="w-64 pl-9"
                        placeholder={t(
                          "Nama, nomor anggota, rekening, atau NIK",
                        )}
                      />
                      <button type="submit" className="sr-only">
                        {t("Cari")}
                      </button>
                    </form>
                  </div>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                      <thead className="bg-secondary/40 text-[11px] uppercase tracking-wider text-muted-foreground">
                        <tr>
                          <th className="p-4 pl-5 font-medium">
                            {t("Anggota")}
                          </th>
                          <th className="p-4 font-medium">{t("KCP asal")}</th>
                          <th className="p-4 pr-5 text-right font-medium">
                            {t("Saldo simpanan")}
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {visibleMembers.map((m) => (
                          <tr
                            key={m.id}
                            className={cn(
                              "border-t",
                              selected === m.id
                                ? "bg-accent"
                                : "hover:bg-accent/50",
                            )}
                          >
                            <td className="p-4 pl-5">
                              <button
                                className="text-left"
                                onClick={() => setSelected(m.id)}
                              >
                                <span className="font-medium underline-offset-4 hover:underline">
                                  {m.name}
                                </span>
                                <span className="mt-1 block font-mono text-[11px] text-muted-foreground">
                                  {m.number} · {t("Rekening")}{" "}
                                  {m.account_number}
                                </span>
                              </button>
                            </td>
                            <td className="p-4 text-xs text-muted-foreground">
                              {m.home_branch}
                            </td>
                            <td className="p-4 pr-5 text-right font-mono text-xs">
                              {money(m.balance)}
                            </td>
                          </tr>
                        ))}
                        {members.length === 0 && (
                          <tr>
                            <td
                              colSpan={3}
                              className="p-12 text-center text-muted-foreground"
                            >
                              <Users className="mx-auto mb-3" />
                              <p>{t("Belum ada anggota yang ditemukan.")}</p>
                              <p className="mt-2 text-xs">
                                {t(
                                  "Daftarkan anggota pertama atau ubah pencarian.",
                                )}
                              </p>
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                  <div className="flex flex-wrap items-center justify-between gap-3 border-t px-5 py-4 text-xs text-muted-foreground">
                    <p role="status">
                      {t("Menampilkan {start}-{end} dari {count} anggota · Halaman {page} dari {pages}", {
                        start: memberRowStart,
                        end: memberRowEnd,
                        count: members.length,
                        page: currentMemberPage,
                        pages: memberPageCount,
                      })}
                    </p>
                    <div className="flex gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={currentMemberPage <= 1}
                        onClick={() => setMemberPage(currentMemberPage - 1)}
                      >
                        <ChevronLeft aria-hidden="true" size={14} />
                        {t("Sebelumnya")}
                      </Button>
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={currentMemberPage >= memberPageCount}
                        onClick={() => setMemberPage(currentMemberPage + 1)}
                      >
                        {t("Berikutnya")}
                        <ChevronRight aria-hidden="true" size={14} />
                      </Button>
                    </div>
                  </div>
                </Card>
                <Card>
                  <CardContent className="pt-6">
                    {member ? (
                      <>
                        <Badge variant="outline" className="mb-4">
                          {member.number}
                        </Badge>
                        <h2 className="font-serif text-2xl">{member.name}</h2>
                        <p className="mt-2 font-mono text-xs text-muted-foreground">
                          {t("NIK")} {member.nik_masked}
                        </p>
                        <dl className="my-6 grid gap-4 text-sm">
                          <div>
                            <dt className="mb-1 text-xs text-muted-foreground">
                              {t("Nomor rekening")}
                            </dt>
                            <dd className="font-mono">
                              {member.account_number}
                            </dd>
                          </div>
                          <div>
                            <dt className="mb-1 text-xs text-muted-foreground">
                              {t("KCP asal")}
                            </dt>
                            <dd>{member.home_branch}</dd>
                          </div>
                          <div>
                            <dt className="mb-1 text-xs text-muted-foreground">
                              {t("Telepon")}
                            </dt>
                            <dd>{member.phone}</dd>
                          </div>
                          <div>
                            <dt className="mb-1 text-xs text-muted-foreground">
                              {t("Alamat")}
                            </dt>
                            <dd>{member.address}</dd>
                          </div>
                        </dl>
                        <div className="border-t pt-4">
                          <p className="text-xs text-muted-foreground">
                            {t("Saldo tersedia")}
                          </p>
                          <p className="mt-2 font-mono text-xl">
                            {money(member.balance)}
                          </p>
                        </div>
                        {user.role === "teller" && (
                          <div className="mt-6 grid gap-2 sm:grid-cols-2">
                            <Button onClick={() => navigate("deposit")} disabled={busy}>
                              <ArrowDownLeft size={15} /> {t("Setor Tunai")}
                            </Button>
                            <Button onClick={() => navigate("withdrawal")} disabled={busy}>
                              <ArrowUpRight size={15} /> {t("Tarik Tunai")}
                            </Button>
                          </div>
                        )}
                      </>
                    ) : (
                      <div className="py-8 text-center">
                        <Users className="mx-auto mb-4 text-muted-foreground" />
                        <h2 className="font-medium">{t("Detail anggota")}</h2>
                        <p className="mt-2 text-xs leading-6 text-muted-foreground">
                          {t(
                            "Pilih nama anggota untuk melihat profil dan saldo simpanannya.",
                          )}
                        </p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </div>
            </>
          )}
          {isCash && (
            <div className="grid max-w-4xl items-start gap-6">
              <Card className="w-full max-w-[440px]">
                <CardContent className="pt-6">
                  <form
                    className="mb-6 grid gap-3"
                    onSubmit={(event) => {
                      event.preventDefault();
                      void lookupAccount();
                    }}
                  >
                    <Field label={t("Nomor rekening")}>
                      <Input
                        aria-label={t("Nomor rekening")}
                        inputMode="numeric"
                        pattern="[1-9][0-9]{9}"
                        minLength={10}
                        maxLength={10}
                        required
                        value={accountInput}
                        disabled={busy}
                        onChange={(event) => {
                          accountLookupVersion.current++;
                          setAccountInput(event.target.value);
                          setSelected(null);
                          setResolvedMember(null);
                          setConfirmCash(false);
                          setError("");
                          setSuccess(null);
                        }}
                        placeholder={t("10 digit nomor rekening")}
                      />
                    </Field>
                    <Button type="submit" variant="outline" disabled={busy}>
                      <Search size={15} />
                      {t("Cari rekening")}
                    </Button>
                  </form>
                  <form
                    id="cash-transaction"
                    className="grid gap-5"
                    onSubmit={(e) => {
                      e.preventDefault();
                      if (!member || busy) return;
                      if (!confirmCash) {
                        setConfirmCash(true);
                        return;
                      }
                      run(async () => {
                        const result = await mutate<{ reference: string }>(
                          "/cash",
                          {
                            member_id: selected,
                            amount: Number(cashAmount),
                            kind: cashKind,
                            note: cashNote,
                          },
                        );
                        setLastReceipt(result.reference);
                        setSuccess({
                          key: "{kind} Rp{amount} untuk {name} berhasil. Referensi: {reference}",
                          kind:
                            cashKind === "deposit" ? "deposit" : "withdrawal",
                          values: {
                            amount: Number(cashAmount),
                            name: member?.name || "",
                            reference: result.reference,
                          },
                        });
                        setCashAmount("");
                        setCashNote("");
                        setConfirmCash(false);
                        await refresh();
                        const updated = await api<Member>(
                          `/members/by-account/${member.account_number}`,
                        );
                        setResolvedMember(updated);
                        setHistoryResult({
                          memberID: selected,
                          actorID: user.id,
                          rows: await api<Transaction[]>(`/members/${selected}/transactions`),
                        });
                      });
                    }}
                  >
                    {member && (
                      <div className="rounded-md bg-secondary/40 p-3 text-sm">
                        <p className="text-xs text-muted-foreground">
                          {member.name} · {member.account_number}
                        </p>
                        <p className="mt-1 font-mono">
                          {money(member.balance)}
                        </p>
                      </div>
                    )}
                    <Field label={t("Nominal (Rp)")}>
                      <AmountInput
                        aria-label={t("Nominal transaksi")}
                        required
                        value={cashAmount}
                        onValueChange={(digits) => {
                          setCashAmount(digits);
                          setConfirmCash(false);
                        }}
                        placeholder="0"
                      />
                    </Field>
                    <Field label={t("Catatan (opsional)")}>
                      <Textarea
                        maxLength={500}
                        value={cashNote}
                        onChange={(e) => {
                          setCashNote(e.target.value);
                          setConfirmCash(false);
                        }}
                        placeholder={t("Keterangan transaksi")}
                      />
                    </Field>
                    <Modal open={confirmCash} onClose={()=>setConfirmCash(false)} blocked={busy}
                      title={t("Konfirmasi") + " " + t(cashKind === "deposit" ? "setoran" : "penarikan")}
                      footer={<><Button type="button" variant="ghost" disabled={busy} onClick={()=>setConfirmCash(false)}>{t("Batal")}</Button><Button type="submit" form="cash-transaction" disabled={busy}>{t("Konfirmasi & simpan transaksi")}</Button></>}>
                      <p>{member?.name} · {member?.account_number} · {money(Number(cashAmount))}</p>
                      <p className="mt-3 text-sm text-muted-foreground">{t("Pastikan nominal dan uang tunai sudah diperiksa.")}</p>
                    </Modal>
                    <Button
                      disabled={busy || user.role !== "teller" || !member}
                    >
                      {busy ? <Loader2 className="animate-spin" /> : null}
                      {confirmCash
                        ? t("Konfirmasi & simpan transaksi")
                        : t("Periksa transaksi")}
                    </Button>
                    {user.role !== "teller" && (
                      <p className="text-xs text-muted-foreground">
                        {t("Pencatatan transaksi dilakukan oleh Teller.")}
                      </p>
                    )}
                  </form>
                  <div className="mt-6 border-t pt-4 text-xs text-muted-foreground">
                    {t("Kas tercatat")} {user.branch_name}
                    <span className="mt-1 block font-mono text-sm text-foreground">
                      {money(meta?.branch_cash || 0)}
                    </span>
                  </div>
                </CardContent>
              </Card>
              <History
                key={`${selected ?? "none"}-${history[0]?.reference ?? "empty"}`}
                rows={history}
                selected={!!selected}
              />
            </div>
          )}
          {tab === "interest" && user.role === "supervisor" && <InterestSettings key={user.id} />}
{tab === "transactions" && <TransactionList />}
          {tab === "loans" && <LoanWorkspace key={user.id} user={user} branches={meta?.branches ?? []} refresh={async () => { await refresh(); }} />}
          <footer className="no-print mt-10 flex flex-wrap items-center justify-between gap-3 border-t pt-5 text-[10px] uppercase tracking-wider text-muted-foreground">
            <span>{t("Credit Union · Layanan anggota")}</span>
            <span className="flex items-center gap-2">
              <ShieldCheck size={13} />
              {t("Akses sesuai kewenangan")}
            </span>
          </footer>
        </main>
      </div>
    </div>
  );
}
function History({
  rows,
  selected,
}: {
  rows: Transaction[];
  selected: boolean;
}) {
  const { t, money, date } = usePreferences();
  const [page, setPage] = useState(1);
  const pageCount = Math.max(1, Math.ceil(rows.length / TABLE_PAGE_SIZE));
  const currentPage = Math.min(page, pageCount);
  const visibleRows = rows.slice(
    (currentPage - 1) * TABLE_PAGE_SIZE,
    currentPage * TABLE_PAGE_SIZE,
  );
  const rowStart = rows.length ? (currentPage - 1) * TABLE_PAGE_SIZE + 1 : 0;
  const rowEnd = Math.min(currentPage * TABLE_PAGE_SIZE, rows.length);
  return (
    <Card className="gap-0 overflow-hidden py-0">
      <div className="border-b p-5">
        <h2 className="text-sm font-semibold">{t("Riwayat transaksi")}</h2>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("Transaksi simpanan terbaru pada rekening anggota")}
        </p>
        <p className="mt-1 text-xs text-muted-foreground">
          {t("Maksimal 100 transaksi terbaru")}
        </p>
      </div>
      {rows.length === 0 ? (
        <div className="p-12 text-center text-sm text-muted-foreground">
          <Wallet className="mx-auto mb-3" />
          {selected
            ? t("Belum ada transaksi.")
            : t("Pilih anggota untuk melihat riwayat.")}
        </div>
      ) : (
        <div className="divide-y">
          {visibleRows.map((transaction) => (
            <div key={transaction.reference} className="flex gap-3 p-5">
              <span className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-secondary">
                {transaction.kind !== "withdrawal" ? (
                  <ArrowDownLeft size={16} />
                ) : (
                  <ArrowUpRight size={16} />
                )}
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap justify-between gap-2">
                  <p className="text-sm font-medium">
                    {t(movementLabel(transaction.kind))}
                  </p>
                  <p className="font-mono text-xs">
                    {movementSign(transaction.kind)}
                    {money(transaction.amount)}
                  </p>
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  {transaction.balance_before == null || transaction.balance_after == null
                    ? t("Saldo historis tidak tersedia")
                    : `${t("Saldo sebelum")}: ${money(transaction.balance_before)} → ${t("Saldo sesudah")}: ${money(transaction.balance_after)}`}
                </p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {transaction.branch} · {transaction.actor === "System" ? t("Sistem") : transaction.actor}
                </p>
                <p className="mt-1 text-[10px] text-muted-foreground">
                  {date(transaction.created_at)}
                </p>
                {transaction.note && (
                  <p className="mt-2 text-xs">{transaction.note}</p>
                )}
                <p className="mt-2 break-all font-mono text-[10px] text-muted-foreground">
                  {transaction.reference}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
      {rows.length > 0 && (
        <div className="flex flex-wrap items-center justify-between gap-3 border-t px-5 py-4 text-xs text-muted-foreground">
          <p role="status">
            {t("Menampilkan {start}-{end} dari {count} transaksi · Halaman {page} dari {pages}", {
              start: rowStart,
              end: rowEnd,
              count: rows.length,
              page: currentPage,
              pages: pageCount,
            })}
          </p>
          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={currentPage <= 1}
              onClick={() => setPage(currentPage - 1)}
            >
              <ChevronLeft aria-hidden="true" size={14} />
              {t("Sebelumnya")}
            </Button>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={currentPage >= pageCount}
              onClick={() => setPage(currentPage + 1)}
            >
              {t("Berikutnya")}
              <ChevronRight aria-hidden="true" size={14} />
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
}
