export interface TableQuery {
  /** Free-text search. Matched case-insensitively against `searchFields`. */
  q?: string;
  /** Column id to sort by. Falls back to `defaultSort` when absent. */
  sort?: string;
  /** Sort direction. Absent means ascending. */
  dir?: "asc" | "desc";
}

interface ApplyTableQueryOptions<T> {
  searchFields: (row: T) => Array<string | null | undefined>;
  sortValue: (row: T, column: string) => string;
  defaultSort: string;
}

/**
 * Client-side search + sort for console data tables. Pure so table routes
 * stay thin: the URL (TanStack Router `validateSearch`) is the source of
 * truth, this derives the visible rows.
 */
export function applyTableQuery<T>(
  rows: readonly T[],
  query: TableQuery,
  { searchFields, sortValue, defaultSort }: ApplyTableQueryOptions<T>,
): T[] {
  const q = query.q?.trim().toLowerCase();
  const filtered = q
    ? rows.filter((row) =>
        searchFields(row).some((f) => f?.toLowerCase().includes(q)),
      )
    : [...rows];

  const column = query.sort ?? defaultSort;
  const dir = query.dir === "desc" ? -1 : 1;
  const collator = new Intl.Collator(undefined, {
    numeric: true,
    sensitivity: "base",
  });
  filtered.sort(
    (a, b) =>
      collator.compare(sortValue(a, column), sortValue(b, column)) * dir,
  );
  return filtered;
}
