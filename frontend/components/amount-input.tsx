"use client";

import { useLayoutEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import { usePreferences } from "@/components/preferences";
import { amountCaret, amountDigits, formatAmountInput } from "@/lib/amount-input";

export function AmountInput({ name, value, onValueChange, required = false }: {
  name?: string;
  value: string;
  onValueChange: (digits: string) => void;
  required?: boolean;
}) {
  const { locale, t } = usePreferences();
  const input = useRef<HTMLInputElement>(null);
  const cursor = useRef<number | null>(null);
  const formatted = formatAmountInput(value, locale);

  useLayoutEffect(() => {
    const element = input.current;
    if (!element) return;
    const valid = value === "" || (Number.isSafeInteger(Number(value)) && Number(value) >= 1 && Number(value) <= 1000000000000);
    element.setCustomValidity(valid ? "" : t("Nominal harus antara Rp1 dan Rp1.000.000.000.000."));
    if (cursor.current !== null) {
      const position = amountCaret(formatted, cursor.current);
      element.setSelectionRange(position, position);
      cursor.current = null;
    }
  }, [value, formatted, locale, t]);

  return <>
    {name && <input type="hidden" name={name} value={value} />}
    <Input ref={input} type="text" inputMode="numeric" autoComplete="off" required={required}
      value={formatted} onChange={event => {
        const digits = amountDigits(event.target.value);
        if (digits === null) return;
        cursor.current = event.target.value.slice(0, event.target.selectionStart ?? 0).replace(/[^0-9]/g, "").length;
        onValueChange(digits);
      }} />
  </>;
}
