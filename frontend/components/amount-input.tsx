"use client";

import { useLayoutEffect, useRef, type ComponentProps } from "react";
import { Input } from "@/components/ui/input";
import { usePreferences } from "@/components/preferences";
import { amountCaret, amountDigits, formatAmountInput, validAmountInput } from "@/lib/amount-input";

export function AmountInput({ name, value, onValueChange, required = false, minimum = 1, ...inputProps }: {
  name?: string;
  value: string;
  onValueChange: (digits: string) => void;
  required?: boolean;
  minimum?: 0 | 1;
} & Pick<ComponentProps<"input">, "aria-label" | "placeholder" | "id">) {
  const { locale, t } = usePreferences();
  const input = useRef<HTMLInputElement>(null);
  const cursor = useRef<number | null>(null);
  const formatted = formatAmountInput(value, locale);

  useLayoutEffect(() => {
    const element = input.current;
    if (!element) return;
    const valid = validAmountInput(value, minimum);
    element.setCustomValidity(valid ? "" : t(minimum === 0 ? "Nominal harus antara Rp0 dan Rp1.000.000.000.000." : "Nominal harus antara Rp1 dan Rp1.000.000.000.000."));
    if (cursor.current !== null) {
      const position = amountCaret(formatted, cursor.current);
      element.setSelectionRange(position, position);
      cursor.current = null;
    }
  }, [value, formatted, locale, minimum, t]);

  return <>
    {name && <input type="hidden" name={name} value={value} />}
    <Input {...inputProps} ref={input} type="text" inputMode="numeric" autoComplete="off" required={required}
      value={formatted} onChange={event => {
        const digits = amountDigits(event.target.value);
        if (digits === null) return;
        cursor.current = event.target.value.slice(0, event.target.selectionStart ?? 0).replace(/[^0-9]/g, "").length;
        onValueChange(digits);
      }} />
  </>;
}
