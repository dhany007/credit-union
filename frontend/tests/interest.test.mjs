import assert from "node:assert/strict";
import { test } from "node:test";
import { rateToBps, nextJakartaDate, movementLabel, movementSign } from "../lib/interest.ts";
import { translate } from "../lib/messages.ts";

test("savings percentages retain exact basis points and reject invalid precision", () => {
  for (const [input, expected] of [["3",300],["4.5",450],["0.01",1],["100",10000],["100.01",null],["-1",null],["3.001",null],["",null]]) {
    assert.equal(rateToBps(input),expected,input);
  }
});
test("policy minimum follows Jakarta midnight", () => {
  assert.equal(nextJakartaDate(new Date("2026-10-23T16:59:59Z")),"2026-10-24");
  assert.equal(nextJakartaDate(new Date("2026-10-23T17:00:00Z")),"2026-10-25");
});
test("interest is positive and explicitly translated in both languages", () => {
  assert.equal(movementSign("interest_credit"),"+");
  assert.equal(movementSign("withdrawal"),"−");
  assert.equal(translate("en",movementLabel("interest_credit")),"Savings interest");
  assert.equal(translate("id",movementLabel("interest_credit")),"Bunga simpanan");
});
