export function amountDigits(input: string): string | null {
  if (!/^[0-9.,\s]*$/.test(input)) return null;
  const digits = input.replace(/[.,\s]/g, "").replace(/^0+(?=\d)/, "");
  return digits.length <= 16 ? digits : null;
}

export function formatAmountInput(digits: string, locale: "id" | "en"): string {
  return digits === "" ? "" : new Intl.NumberFormat(locale === "id" ? "id-ID" : "en-US", { maximumFractionDigits: 0 }).format(Number(digits));
}

export function amountCaret(formatted: string, digitsBefore: number): number {
  if (!digitsBefore) return 0;
  let count = 0;
  for (let i = 0; i < formatted.length; i++) {
    if (/[0-9]/.test(formatted[i]) && ++count === digitsBefore) return i + 1;
  }
  return formatted.length;
}
