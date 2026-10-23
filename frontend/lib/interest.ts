export function rateToBps(value: string): number | null {
  if (!/^\d{1,3}(\.\d{1,2})?$/.test(value.trim())) return null;
  const [whole, fraction = ""] = value.trim().split(".");
  const bps = Number(whole) * 100 + Number(fraction.padEnd(2, "0"));
  return bps <= 10000 ? bps : null;
}
export function nextJakartaDate(now = new Date()): string {
  return new Date(now.getTime() + 7 * 3600000 + 86400000).toISOString().slice(0, 10);
}
export function movementLabel(kind: string): "Setoran tunai" | "Penarikan tunai" | "Bunga simpanan" {
  return kind === "interest_credit" ? "Bunga simpanan" : kind === "withdrawal" ? "Penarikan tunai" : "Setoran tunai";
}
export function movementSign(kind: string): "+" | "−" {
  return kind === "withdrawal" ? "−" : "+";
}
