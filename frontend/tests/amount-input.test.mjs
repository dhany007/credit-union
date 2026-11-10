import assert from "node:assert/strict";
import { test } from "node:test";
import { amountDigits, formatAmountInput, amountCaret, validAmountInput } from "../lib/amount-input.ts";

test("amount input follows ID/EN grouping without changing the canonical API value", () => {
  assert.equal(formatAmountInput("1000000", "id"), "1.000.000");
  assert.equal(formatAmountInput("1000000", "en"), "1,000,000");
  assert.equal(amountDigits("1.000.000"), "1000000");
  assert.equal(amountDigits("1,000,000"), "1000000");
  assert.equal(Number(amountDigits("1.000.000")), 1000000);
});
test("empty, pasted, leading-zero and maximum amounts retain integer digits", () => {
  assert.equal(formatAmountInput("", "id"), "");
  assert.equal(amountDigits(""), "");
  assert.equal(amountDigits("0001000"), "1000");
  assert.equal(amountDigits(" 1 000 000 "), "1000000");
  assert.equal(amountDigits("1.000.000.000.000"), "1000000000000");
});
test("negative amounts, exponent notation and text are not silently turned into positive money", () => {
  for (const value of ["-1000", "1e6", "abc", "10000000000000000"]) assert.equal(amountDigits(value), null);
});
test("editing a grouped amount preserves the caret by digit position", () => {
  assert.equal(amountCaret("1.234.567", 2), 3);
  assert.equal(amountCaret("1,234,567", 5), 7);
  assert.equal(amountCaret("1.000", 0), 0);
  assert.equal(amountCaret("", 0), 0);
});

test("zero is allowed only for lower band boundaries and blank upper bounds remain optional", () => {
  assert.equal(validAmountInput("0", 0), true);
  assert.equal(validAmountInput("0"), false);
  assert.equal(validAmountInput(""), true);
  assert.equal(validAmountInput("1000000000000"), true);
  for (const value of ["1000000000001", "-1", "1.5", "1,000", "abc"]) assert.equal(validAmountInput(value, 0), false);
});
