import assert from "node:assert/strict";
import { test } from "node:test";
import { applicationStage, formatLoanTimestamp, jakartaToday, payoutBody, repaymentBody, loanListReference } from "../lib/loans.ts";
import { submitLoanCommand } from "../lib/loan-command.ts";
test("business date crosses midnight in Jakarta rather than the client zone", () => {
 assert.equal(jakartaToday(new Date("2026-11-10T17:01:00Z")), "2026-11-11");
});
test("payout binds every server-approved field without recalculating rates", () => {
 assert.deepEqual(payoutBody({ approval_revision: 2, policy_id: 9, annual_rate_bps: 450, amount: 5000000, tenor_months: 12, payout_date: "2026-11-10" }, "KTP verified"), {
 expected_approval_revision: 2, expected_policy_id: 9, expected_rate_bps: 450, expected_principal: 5000000, expected_tenor_months: 12, expected_payout_date: "2026-11-10", recipient_verified: true, agreement_signed: true, conditions_verified: true, verification_note: "KTP verified", note: ""
 });
});
function storage() { const values = new Map(); return { values, getItem: k => values.get(k) ?? null, setItem: (k,v) => values.set(k,v), removeItem: k => values.delete(k) }; }
const command = { path: "/loan-applications/5/disburse", payload: '{"expected_principal":1000000}', key: "original-key" };
test("uncertain failure survives reload and replay uses the original body/key", async () => {
 const s=storage();
 await assert.rejects(submitLoanCommand(command,s,"pending",async c=>{ assert.deepEqual(JSON.parse(s.values.get("pending")),command); assert.equal(c.key,"original-key"); throw Error("Failed to fetch"); }));
 const restored=JSON.parse(s.values.get("pending"));
 const receipt=await submitLoanCommand(restored,s,"pending",async c=>{ assert.deepEqual(c,command); return { reference:"TX-once" }; });
 assert.equal(receipt.reference,"TX-once"); assert.equal(s.values.has("pending"),false);
});
for(const status of [401,403,408,429,500,503]) test("HTTP "+status+" keeps the original command", async () => {const s=storage();await assert.rejects(submitLoanCommand(command,s,"pending",async()=>{throw Object.assign(Error("retry"),{status});}));assert.deepEqual(JSON.parse(s.values.get("pending")),command);});
for(const status of [400,404,409,422]) test("HTTP "+status+" permits a fresh review after definitive rejection", async () => {const s=storage();await assert.rejects(submitLoanCommand(command,s,"pending",async()=>{throw Object.assign(Error("rejected"),{status});}));assert.equal(s.values.has("pending"),false);});
test("storage failure prevents sending a cash request", async () => {let sent=false;await assert.rejects(submitLoanCommand(command,{getItem(){return null;},setItem(){throw Error("blocked");},removeItem(){}},"pending",async()=>{sent=true;}));assert.equal(sent,false);});

test("an undisbursed application keeps its PJ reference even with a null loan_number",()=>{assert.equal(loanListReference({application_number:"PJ-25",loan_number:null}),"PJ-25");assert.equal(loanListReference({loan_number:"LN-1"}),"LN-1");});

test("application timestamps use Jakarta and undecided rows show a dash", () => {
 assert.equal(formatLoanTimestamp(null, "id"), "—");
 const id = formatLoanTimestamp("2026-11-10T17:30:00Z", "id");
 const en = formatLoanTimestamp("2026-11-10T17:30:00Z", "en");
 assert.match(id, /11 Nov 2026/);
 assert.match(id, /00[.:]30 WIB$/);
 assert.match(en, /11 Nov 2026/);
 assert.match(en, /00:30 WIB$/);
});

test("application display status separates approved and actually disbursed loans", () => {
 assert.equal(applicationStage({status:"submitted",linked_loan_id:null}), "Menunggu persetujuan");
 assert.equal(applicationStage({status:"approved",linked_loan_id:null}), "Menunggu pencairan");
 assert.equal(applicationStage({status:"approved",linked_loan_id:42}), "Sudah dicairkan");
 assert.equal(applicationStage({status:"rejected",linked_loan_id:null}), "Ditolak");
});

test("late payout replay cannot erase a newer pending payout", async () => {
 const s = storage();
 let finishOriginal;
 let finishReplay;
 const original = submitLoanCommand(command, s, "pending", () => new Promise(resolve => { finishOriginal = resolve; }));
 const replay = submitLoanCommand(command, s, "pending", () => new Promise(resolve => { finishReplay = resolve; }));
 finishOriginal({ reference: "TX-original" });
 await original;
 const next = { ...command, path: "/loan-applications/6/disburse", key: "next-key" };
 await assert.rejects(submitLoanCommand(next, s, "pending", async () => { throw Error("Response lost"); }));
 finishReplay({ reference: "TX-original" });
 await replay;
 assert.deepEqual(JSON.parse(s.getItem("pending")), next);
});

test("payment uses the reviewed amount and revisions", () => {
 assert.deepEqual(repaymentBody({ amount: 12345, expected_payment_revision: 4, current_schedule_version_id: 8 }, "cash"), {
  amount: 12345, expected_payment_revision: 4, expected_schedule_version_id: 8, note: "cash",
 });
});
