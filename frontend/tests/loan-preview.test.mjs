import assert from "node:assert/strict";
import { test } from "node:test";
import { scheduleLoanPreview } from "../lib/loan-preview.ts";

test("typing is debounced and obsolete scheduled requests never start", async t => {
  t.mock.timers.enable({apis:["setTimeout"]});
  let requests=0; const rows=[];
  const cancel=scheduleLoanPreview(async()=>{requests++;return {amount:100};},v=>rows.push(v),()=>assert.fail());
  t.mock.timers.tick(349); assert.equal(requests,0);
  cancel(); t.mock.timers.tick(1); await Promise.resolve();
  assert.equal(requests,0); assert.deepEqual(rows,[]);
});
test("a current preview is delivered after the typing delay", async t => {
  t.mock.timers.enable({apis:["setTimeout"]});
  let requests=0; const rows=[];
  const cancel=scheduleLoanPreview(async()=>{requests++;return {amount:200};},v=>rows.push(v),()=>assert.fail());
  t.mock.timers.tick(350); await Promise.resolve();
  assert.equal(requests,1);assert.deepEqual(rows,[{amount:200}]);cancel();
});
test("late results from earlier terms cannot replace the latest schedule", async t => {
  t.mock.timers.enable({apis:["setTimeout"]});
  let oldResolve,newResolve,oldSignal; const rows=[];
  const cancelOld=scheduleLoanPreview(signal=>{oldSignal=signal;return new Promise(resolve=>{oldResolve=resolve;});},v=>rows.push(v),()=>assert.fail());
  t.mock.timers.tick(350);cancelOld();assert.equal(oldSignal.aborted,true);
  const cancelNew=scheduleLoanPreview(()=>new Promise(resolve=>{newResolve=resolve;}),v=>rows.push(v),()=>assert.fail());
  t.mock.timers.tick(350);newResolve({amount:200});await Promise.resolve();
  oldResolve({amount:100});await Promise.resolve();
  assert.deepEqual(rows,[{amount:200}]);cancelNew();
});
test("obsolete failures stay silent; a current failure is exposed for retry", async t => {
  t.mock.timers.enable({apis:["setTimeout"]});
  let rejectOld;const errors=[];
  const cancel=scheduleLoanPreview(()=>new Promise((_,reject)=>{rejectOld=reject;}),()=>assert.fail(),e=>errors.push(e.message));
  t.mock.timers.tick(350);cancel();rejectOld(Error("old"));await Promise.resolve();
  const cancelNew=scheduleLoanPreview(async()=>{throw Error("current");},()=>assert.fail(),e=>errors.push(e.message));
  t.mock.timers.tick(350);await Promise.resolve();
  assert.deepEqual(errors,["current"]);cancelNew();
});
