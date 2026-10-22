import { useQuery } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { z } from "zod";

import { CodeBlock, CreateDeviceSplitButton } from "@/components/console";
import { SessionsTable, type SessionFilters } from "@/components/sessions-table";
import { useTRPC } from "@/utils/trpc";

const searchSchema = z.object({
  q: z.string().optional(),
  status: z.string().optional(),
  sort: z.string().optional(),
  dir: z.enum(["asc", "desc"]).optional(),
});

export const Route = createFileRoute("/_auth/simulators")({
  component: RouteComponent,
  validateSearch: (search) => searchSchema.parse(search),
  head: () => ({
    meta: [{ title: "Simulators — Sidekick Console" }],
  }),
});

const COMMANDS = `sidekick ios create --device "iPhone 17"   # or Pixel_7 via: sidekick android create
sidekick ios get <id>                  # poll until running
sidekick ios ax-tree <id>              # read the screen
sidekick android ui-tree <id>          # same verbs, both platforms`;

function RouteComponent() {
  const trpc = useTRPC();
  const navigate = Route.useNavigate();
  const search = Route.useSearch();
  const sims = useQuery(trpc.devices.listSims.queryOptions());
  const emus = useQuery(trpc.devices.listEmus.queryOptions());
  const loading = sims.isPending || emus.isPending;
  const sessions = [
    ...(sims.data ?? []).map((s) => ({ id: s.id, label: s.device, kind: "simulator" as const, status: s.status })),
    ...(emus.data ?? []).map((e) => ({ id: e.id, label: e.profile, kind: "emulator" as const, status: e.status })),
  ];
  const failed = sims.isError || emus.isError;
  const filters: SessionFilters = {
    q: search.q ?? "",
    status: search.status ?? "all",
    sort: search.sort === "label" || search.sort === "status" ? search.sort : "",
    dir: search.dir ?? "asc",
  };

  return (
    <div className="mx-auto w-full max-w-3xl">
      <p className="font-jakarta text-xs font-bold tracking-[0.18em] text-nest uppercase">
        Devices
      </p>
      <h1 className="mt-3 text-4xl font-extrabold tracking-tight text-white md:text-5xl">
        Simulators<span className="text-nest">.</span>
      </h1>
      <p className="mt-4 max-w-xl text-[15px] leading-relaxed text-zinc-400">
        {loading
          ? "Checking for sessions…"
          : sessions.length > 0
            ? `${sessions.length} session${sessions.length === 1 ? "" : "s"} in this org — share links open in any browser, no login needed.`
            : "No sessions yet. Create one below, boot it from the CLI, or let your agent drive — share links open in any browser, no login needed."}
      </p>
      {sessions.length > 0 && !loading && (
        <SessionsTable
          sessions={sessions}
          filters={filters}
          onFiltersChange={(next) =>
            navigate({ search: { q: next.q || undefined, status: next.status === "all" ? undefined : next.status, sort: next.sort || undefined, dir: next.dir }, replace: true })
          }
        />
      )}
      {failed && (
        <p className="mt-6 text-sm text-zinc-500">
          Couldn't reach the device service — use the CLI commands below instead.
        </p>
      )}
      <div className="mt-6">
        <CreateDeviceSplitButton caption="Android ready now · iOS coming soon — or paste the commands below." />
      </div>
      <div className="mt-6">
        <CodeBlock lines={COMMANDS} title="terminal" />
      </div>
      <p className="mt-6 text-sm text-zinc-500">
        New here? Start at{" "}
        <Link to="/console" className="underline hover:text-white">
          Get started
        </Link>{" "}
        to create your first API key.
      </p>
    </div>
  );
}
