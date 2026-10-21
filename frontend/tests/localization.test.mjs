import assert from "node:assert/strict";
import { test } from "node:test";
import { localizedError } from "../lib/api-errors.ts";
import { messages, translate } from "../lib/messages.ts";

test("EN/ID translate transaction date controls", () => {
  assert.equal(translate("id", "30 hari terakhir"), "30 hari terakhir");
  assert.equal(translate("en", "30 hari terakhir"), "Last 30 days");
  assert.equal(translate("en", "Rentang khusus"), "Custom range");
  assert.equal(translate("en", "Tanggal akhir"), "End date");
});

test("page counts follow locale while account numbers remain strings", () => {
  assert.equal(translate("id", "Halaman {page} · {count} transaksi", { page: 1, count: 1000 }), "Halaman 1 · 1.000 transaksi");
  assert.equal(translate("en", "Halaman {page} · {count} transaksi", { page: 1, count: 1000 }), "Page 1 · 1,000 transactions");
  assert.equal(translate("en", "Rekening: {account}", { account: "1000000001" }), "Account: 1000000001");
});

test("EN/ID messages preserve the same interpolation fields", () => {
  const placeholders = (value) => [...value.matchAll(/\{(\w+)\}/g)].map((match) => match[1]).sort();
  for (const [id, en] of Object.entries(messages)) {
    assert.ok(en.trim(), `Missing English translation: ${id}`);
    assert.deepEqual(placeholders(en), placeholders(id), `Translation fields differ: ${id}`);
  }
});

test("missing interpolation values remain visible rather than becoming undefined", () => {
  assert.equal(translate("en", "Rekening: {account}"), "Account: {account}");
});

test("public validation errors localize to Indonesian and preserve English", () => {
  const message = "Invalid date filter. Check period and start/end dates";
  assert.equal(localizedError("id", message), "Filter tanggal tidak valid. Periksa periode dan tanggal mulai/akhir");
  assert.equal(localizedError("en", message), message);
  assert.equal(localizedError("id", "Invalid pagination cursor. Apply the filters again"), "Halaman tidak valid. Terapkan kembali filter");
});

for (const message of ["SELECT private_data FROM members", "toString", "constructor", "__proto__"]) {
  test(`unrecognized API error is hidden in both languages: ${message}`, () => {
    assert.equal(localizedError("id", message), "Terjadi kesalahan. Silakan coba lagi.");
    assert.equal(localizedError("en", message), "Something went wrong. Please try again.");
  });
}
