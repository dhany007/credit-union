import assert from "node:assert/strict";
import { test } from "node:test";
import { api } from "../lib/api.ts";

const unreachable = "The server could not be reached. Please try again.";
const failed = "Operation failed. Please retry with the same reference.";

test("API returns successful JSON and disables caching", async (t) => {
  const payload = { account_number: "1000000001", balance: 50000 };
  t.mock.method(globalThis, "fetch", async (url, options) => {
    assert.equal(url, "/api/members/by-account/1000000001");
    assert.equal(options.cache, "no-store");
    assert.equal(new Headers(options.headers).get("Content-Type"), "application/json");
    return Response.json(payload);
  });
  assert.deepEqual(await api("/members/by-account/1000000001", { cache: "force-cache" }), payload);
});

for (const [label, headers] of [
  ["object", { "Idempotency-Key": "synthetic-request-key", "Content-Type": "application/custom+json" }],
  ["Headers", new Headers({ "Idempotency-Key": "synthetic-request-key", "Content-Type": "application/custom+json" })],
  ["tuple array", [["Idempotency-Key", "synthetic-request-key"], ["Content-Type", "application/custom+json"]]],
]) {
  test(`API preserves ${label} headers and mutation options`, async (t) => {
    t.mock.method(globalThis, "fetch", async (_url, options) => {
      const actual = new Headers(options.headers);
      assert.equal(actual.get("Idempotency-Key"), "synthetic-request-key");
      assert.equal(actual.get("Content-Type"), "application/custom+json");
      assert.equal(options.method, "POST");
      assert.equal(options.body, '{"amount":50000}');
      return Response.json({ reference: "TX-synthetic" });
    });
    assert.deepEqual(await api("/cash", { method: "POST", headers, body: '{"amount":50000}' }), { reference: "TX-synthetic" });
  });
}

test("API preserves a public HTTP validation error", async (t) => {
  t.mock.method(globalThis, "fetch", async () => Response.json({ error: "Invalid account number. Enter exactly 10 digits" }, { status: 400 }));
  await assert.rejects(api("/transactions/search?account_number=123"), { message: "Invalid account number. Enter exactly 10 digits" });
});

for (const payload of [null, {}, { error: { sql: "internal details" } }]) {
  test(`API uses a safe failure for malformed error payload ${JSON.stringify(payload)}`, async (t) => {
    t.mock.method(globalThis, "fetch", async () => Response.json(payload, { status: 500 }));
    await assert.rejects(api("/transactions/search"), { message: failed });
  });
}

for (const status of [200, 502]) {
  test(`API rejects non-JSON HTTP ${status} instead of reporting success`, async (t) => {
    t.mock.method(globalThis, "fetch", async () => new Response("<html>upstream failure</html>", { status }));
    await assert.rejects(api("/transactions/search"), { message: unreachable });
  });
}

test("API forwards cancellation and preserves AbortError", async (t) => {
  const controller = new AbortController();
  controller.abort();
  t.mock.method(globalThis, "fetch", async (_url, options) => {
    assert.equal(options.signal, controller.signal);
    throw new DOMException("Synthetic cancellation", "AbortError");
  });
  await assert.rejects(api("/transactions/search", { signal: controller.signal }), { name: "AbortError" });
});

test("API propagates transport failures without retrying mutations", async (t) => {
  const fetchMock = t.mock.method(globalThis, "fetch", async () => { throw new TypeError("Failed to fetch"); });
  await assert.rejects(api("/cash", { method: "POST" }), { message: "Failed to fetch" });
  assert.equal(fetchMock.mock.callCount(), 1);
});

test("API preserves cancellation while reading the response body", async (t) => {
  const response = Response.json({});
  t.mock.method(response, "json", async () => { throw new DOMException("Synthetic body cancellation", "AbortError"); });
  t.mock.method(globalThis, "fetch", async () => response);
  await assert.rejects(api("/transactions/search"), { name: "AbortError" });
});
