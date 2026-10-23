import { cn } from "@sidekick/ui/lib/utils";
import { ArrowDownIcon, ArrowUpIcon, ChevronsUpDownIcon } from "lucide-react";
import {
  Cell,
  Column,
  Row,
  type SortDescriptor,
  Table,
  TableBody,
  TableHeader,
} from "react-aria-components";

export interface DataTableColumn<T> {
  /** Stable id. Used as the RAC sort key and reflected in the URL. */
  id: string;
  header: string;
  allowsSorting?: boolean;
  className?: string;
  render: (row: T) => React.ReactNode;
}

interface DataTableProps<T> {
  label: string;
  columns: DataTableColumn<T>[];
  rows: T[];
  keyOf: (row: T) => string;
  sortDescriptor: SortDescriptor;
  onSortChange: (descriptor: SortDescriptor) => void;
  /** Shown when `rows` is empty (e.g. filters match nothing). */
  emptyState?: React.ReactNode;
  className?: string;
}

function SortIcon({
  direction,
}: {
  direction: "ascending" | "descending" | undefined;
}) {
  if (direction === "ascending")
    return <ArrowUpIcon className="size-3.5" aria-hidden />;
  if (direction === "descending")
    return <ArrowDownIcon className="size-3.5" aria-hidden />;
  return <ChevronsUpDownIcon className="size-3.5 opacity-40" aria-hidden />;
}

/**
 * Accessible data table (React Aria) in the console's dark shadcn style.
 * Sorting/filtering/search state lives outside — callers keep it in the
 * URL via TanStack Router `validateSearch` so views are shareable.
 */
export function DataTable<T>({
  label,
  columns,
  rows,
  keyOf,
  sortDescriptor,
  onSortChange,
  emptyState,
  className,
}: DataTableProps<T>) {
  return (
    <Table
      aria-label={label}
      sortDescriptor={sortDescriptor}
      onSortChange={onSortChange}
      data-slot="data-table"
      className={cn("w-full border-collapse text-sm", className)}
    >
      <TableHeader>
        {columns.map((col) => (
          <Column
            key={col.id}
            id={col.id}
            isRowHeader={col.id === columns[0]?.id}
            allowsSorting={col.allowsSorting}
            data-slot="data-table-column"
            className={cn(
              "border-white/10 border-b px-3 py-2 text-left align-middle font-semibold text-xs text-zinc-400 outline-none",
              col.allowsSorting &&
                "cursor-pointer select-none hover:text-zinc-200 focus-visible:ring-1 focus-visible:ring-ring/50 data-[hovered]:text-zinc-200",
              col.className,
            )}
          >
            {({ allowsSorting, sortDirection }) => (
              <span className="inline-flex items-center gap-1.5">
                {col.header}
                {allowsSorting && <SortIcon direction={sortDirection} />}
              </span>
            )}
          </Column>
        ))}
      </TableHeader>
      <TableBody
        items={rows}
        renderEmptyState={
          emptyState
            ? () => <div className="px-3 py-10 text-center">{emptyState}</div>
            : undefined
        }
      >
        {(row) => (
          <Row
            id={keyOf(row)}
            data-slot="data-table-row"
            className="border-white/5 border-b outline-none last:border-0 focus-visible:ring-1 focus-visible:ring-ring/50 focus-visible:ring-inset data-[hovered]:bg-white/[0.04]"
          >
            {columns.map((col) => (
              <Cell
                key={col.id}
                data-slot="data-table-cell"
                className={cn(
                  "px-3 py-2.5 align-middle text-zinc-200",
                  col.className,
                )}
              >
                {col.render(row)}
              </Cell>
            ))}
          </Row>
        )}
      </TableBody>
    </Table>
  );
}

export type { SortDescriptor };
