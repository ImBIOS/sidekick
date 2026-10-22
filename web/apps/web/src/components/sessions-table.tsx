import {
  columnVisibilityFeature,
  createColumnHelper,
  createSortedRowModel,
  FlexRender,
  rowSortingFeature,
  tableFeatures,
  useTable,
  type SortingState,
} from "@tanstack/react-table";
import { ArrowUpDownIcon, SearchIcon } from "lucide-react";
import * as React from "react";

import { Badge } from "@sidekick/ui/components/badge";
import { Button } from "@sidekick/ui/components/button";
import { Input } from "@sidekick/ui/components/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@sidekick/ui/components/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@sidekick/ui/components/table";

export type DeviceSession = {
  id: string;
  label: string;
  kind: "simulator" | "emulator";
  status: string;
};

export type SessionFilters = {
  q: string;
  status: string;
  sort: "label" | "status" | "";
  dir: "asc" | "desc";
};

const features = tableFeatures({
  columnVisibilityFeature,
  rowSortingFeature,
  sortedRowModel: createSortedRowModel(),
});

const columnHelper = createColumnHelper<typeof features, DeviceSession>();

const columns = columnHelper.columns([
  columnHelper.accessor("label", {
    header: ({ column }) => (
      <Button
        variant="ghost"
        onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        className="-ml-3 text-zinc-300 hover:text-white"
      >
        Session
        <ArrowUpDownIcon className="size-3.5" />
      </Button>
    ),
    cell: ({ row }) => (
      <div className="min-w-0">
        <p className="truncate text-sm font-semibold text-white">{row.original.label}</p>
        <p className="truncate font-mono text-xs text-zinc-500">
          {row.original.kind} · {row.original.id}
        </p>
      </div>
    ),
  }),
  columnHelper.accessor("kind", {
    header: "Kind",
    enableSorting: false,
    cell: ({ row }) => <span className="text-sm text-zinc-400">{row.original.kind}</span>,
  }),
  columnHelper.accessor("status", {
    header: ({ column }) => (
      <Button
        variant="ghost"
        onClick={() => column.toggleSorting(column.getIsSorted() === "asc")}
        className="-ml-3 text-zinc-300 hover:text-white"
      >
        Status
        <ArrowUpDownIcon className="size-3.5" />
      </Button>
    ),
    cell: ({ row }) => (
      <Badge variant="outline" className="border-nest/30 bg-nest/10 font-mono text-nest">
        {row.original.status}
      </Badge>
    ),
  }),
]);

/** Sessions table on the shadcn data-table pattern: sortable headers, a
 * text search and a status filter. Search/filter run over the rows while
 * sorting is table-driven; all state lives in the URL
 * (?q=&status=&sort=&dir=) so views are shareable and survive reloads. */
export function SessionsTable({
  sessions,
  filters,
  onFiltersChange,
}: {
  sessions: DeviceSession[];
  filters: SessionFilters;
  onFiltersChange: (next: SessionFilters) => void;
}) {
  const sorting: SortingState = React.useMemo(
    () => (filters.sort ? [{ id: filters.sort, desc: filters.dir === "desc" }] : []),
    [filters.sort, filters.dir],
  );

  const filtered = React.useMemo(() => {
    const q = filters.q.trim().toLowerCase();
    return sessions.filter(
      (s) =>
        (filters.status === "all" || s.status === filters.status) &&
        (q === "" ||
          s.label.toLowerCase().includes(q) ||
          s.id.toLowerCase().includes(q) ||
          s.kind.includes(q)),
    );
  }, [sessions, filters.q, filters.status]);

  const table = useTable({
    features,
    data: filtered,
    columns,
    state: { sorting },
    getRowId: (row) => row.id,
    onSortingChange: (updater) => {
      const next = typeof updater === "function" ? updater(sorting) : updater;
      const [first] = next;
      onFiltersChange({
        ...filters,
        sort: (first?.id as SessionFilters["sort"]) ?? "",
        dir: first?.desc ? "desc" : "asc",
      });
    },
  });

  const statuses = React.useMemo(
    () => [...new Set(sessions.map((s) => s.status))].sort(),
    [sessions],
  );

  // Local input state so keystrokes don't race the URL replace-navigations;
  // commits to the URL on blur, Enter, or after a short pause.
  const [draftQ, setDraftQ] = React.useState(filters.q);
  React.useEffect(() => {
    setDraftQ(filters.q);
  }, [filters.q]);
  const commitTimer = React.useRef<ReturnType<typeof setTimeout> | null>(null);
  React.useEffect(
    () => () => {
      if (commitTimer.current) clearTimeout(commitTimer.current);
    },
    [],
  );
  const commitQ = (value: string) => {
    if (commitTimer.current) clearTimeout(commitTimer.current);
    onFiltersChange({ ...filters, q: value });
  };
  const queueQCommit = (value: string) => {
    setDraftQ(value);
    if (commitTimer.current) clearTimeout(commitTimer.current);
    commitTimer.current = setTimeout(() => commitQ(value), 400);
  };

  return (
    <div className="mt-6 space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative min-w-52 flex-1">
          <SearchIcon className="absolute top-1/2 left-3 size-4 -translate-y-1/2 text-zinc-500" />
          <Input
            value={draftQ}
            onChange={(e) => queueQCommit(e.target.value)}
            onBlur={(e) => commitQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") commitQ((e.target as HTMLInputElement).value);
              if (e.key === "Escape") queueQCommit("");
            }}
            placeholder="Search sessions…"
            className="pl-9"
            aria-label="Search sessions"
          />
        </div>
        <Select
          value={filters.status}
          onValueChange={(status) => onFiltersChange({ ...filters, status: status ?? "all" })}
        >
          <SelectTrigger className="w-40" aria-label="Filter by status">
            <SelectValue placeholder="Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All statuses</SelectItem>
            {statuses.map((s) => (
              <SelectItem key={s} value={s}>
                {s}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      <div className="overflow-hidden rounded-xl border border-white/10">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id} className="border-white/10 hover:bg-transparent">
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id} className="text-zinc-500">
                    {header.isPlaceholder ? null : <FlexRender header={header} />}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.length > 0 ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id} className="border-white/10 bg-zinc-950 hover:bg-white/5">
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      <FlexRender cell={cell} />
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow className="border-white/10 hover:bg-transparent">
                <TableCell colSpan={columns.length} className="py-8 text-center text-sm text-zinc-500">
                  No sessions match these filters.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <p className="text-xs text-zinc-600" aria-live="polite">
        {table.getRowModel().rows.length} of {sessions.length} sessions
      </p>
    </div>
  );
}
