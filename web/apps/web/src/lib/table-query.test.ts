import { describe, expect, test } from "bun:test";

import { applyTableQuery } from "./table-query";

interface Row {
  id: string;
  label: string;
  kind: "simulator" | "emulator";
  status: string;
}

const rows: Row[] = [
  { id: "b2", label: "Pixel_7", kind: "emulator", status: "running" },
  { id: "a1", label: "iPhone 17", kind: "simulator", status: "stopped" },
  { id: "c3", label: "iPhone 17 Pro", kind: "simulator", status: "running" },
];

const opts = {
  searchFields: (r: Row) => [r.label, r.id, r.status, r.kind],
  sortValue: (r: Row, column: string) =>
    column === "kind" ? r.kind : column === "status" ? r.status : r.label,
  defaultSort: "label",
};

describe("applyTableQuery", () => {
  test("defaults to ascending sort by the default column", () => {
    expect(applyTableQuery(rows, {}, opts).map((r) => r.id)).toEqual([
      "a1",
      "c3",
      "b2",
    ]);
  });

  test("sorts descending when dir is desc", () => {
    expect(
      applyTableQuery(rows, { dir: "desc" }, opts).map((r) => r.id),
    ).toEqual(["b2", "c3", "a1"]);
  });

  test("sorts by another column", () => {
    // emulator < simulator; stable within equal keys
    expect(
      applyTableQuery(rows, { sort: "kind" }, opts).map((r) => r.id),
    ).toEqual(["b2", "a1", "c3"]);
  });

  test("search matches across fields, case-insensitively", () => {
    expect(
      applyTableQuery(rows, { q: "iphone" }, opts).map((r) => r.id),
    ).toEqual(["a1", "c3"]);
    expect(
      applyTableQuery(rows, { q: "RUNNING" }, opts).map((r) => r.id),
    ).toEqual(["c3", "b2"]);
    expect(applyTableQuery(rows, { q: "a1" }, opts).map((r) => r.id)).toEqual([
      "a1",
    ]);
  });

  test("blank query returns all rows", () => {
    expect(applyTableQuery(rows, { q: "   " }, opts)).toHaveLength(3);
  });

  test("does not mutate the input array", () => {
    const input = [...rows];
    applyTableQuery(input, { sort: "status", dir: "desc" }, opts);
    expect(input.map((r) => r.id)).toEqual(["b2", "a1", "c3"]);
  });
});
