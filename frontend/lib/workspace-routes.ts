export const menuRoutes = {
  members: "/anggota",
  deposit: "/setor-tunai",
  withdrawal: "/tarik-tunai",
  loans: "/pinjaman/pengajuan",
  transactions: "/transaksi",
  interest: "/bunga-simpanan",
} as const;

export const loanRoutes = {
  applications: menuRoutes.loans,
  accounts: "/pinjaman/daftar",
  policies: "/pinjaman/tarif",
} as const;

export const newApplicationRoute = loanRoutes.applications + "/baru";

export function workspaceRoute(path: string) {
  for (const [menu, href] of Object.entries(menuRoutes)) {
    if (menu !== "loans" && path === href) {
      return { tab: menu, loanSection: "applications" as const, selected: null, intake: false };
    }
  }
  for (const section of ["applications", "accounts", "policies"] as const) {
    const href = loanRoutes[section];
    if (path === href || (section === "applications" && path === newApplicationRoute)) {
      return { tab: "loans", loanSection: section, selected: null, intake: path === newApplicationRoute };
    }
    if (section !== "policies" && path.startsWith(href + "/")) {
      const segment = path.slice(href.length + 1);
      const id = Number(segment);
      if (/^[1-9][0-9]*$/.test(segment) && Number.isSafeInteger(id)) {
        return { tab: "loans", loanSection: section, selected: id, intake: false };
      }
    }
  }
  return null;
}
