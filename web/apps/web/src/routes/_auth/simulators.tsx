import { Button } from "@sidekick/ui/components/button";
import {
	DataTable,
	type SortDescriptor,
} from "@sidekick/ui/components/data-table";
import { Input } from "@sidekick/ui/components/input";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import { useEffect, useRef, useState } from "react";
import { z } from "zod";
import { CodeBlock, CreateDeviceSplitButton } from "@/components/console";
import { applyTableQuery } from "@/lib/table-query";
import { useTRPC } from "@/utils/trpc";

const searchSchema = z.object({
	q: z.string().optional(),
	// Lenient: unknown values (hand-edited URLs) fall back to defaults
	// instead of throwing into the route error boundary.
	sort: z.enum(["device", "kind", "status"]).optional().catch(undefined),
	dir: z.enum(["asc", "desc"]).optional().catch(undefined),
	kind: z.enum(["simulator", "emulator"]).optional().catch(undefined),
});

export const Route = createFileRoute("/_auth/simulators")({
	validateSearch: (search) => searchSchema.parse(search),
	component: RouteComponent,
	head: () => ({
		meta: [{ title: "Simulators — Sidekick Console" }],
	}),
});

const COMMANDS = `sidekick ios create --device "iPhone 17"   # or Pixel_7 via: sidekick android create
sidekick ios get <id>                  # poll until running
sidekick ios ax-tree <id>              # read the screen
sidekick android ui-tree <id>          # same verbs, both platforms`;

interface SessionRow {
	id: string;
	label: string;
	kind: "simulator" | "emulator";
	status: string;
}

const SEARCH_DEBOUNCE_MS = 200;

function RouteComponent() {
	const { q, sort, dir, kind } = Route.useSearch();
	const navigate = useNavigate({ from: Route.fullPath });
	const trpc = useTRPC();
	const sims = useQuery(trpc.devices.listSims.queryOptions());
	const emus = useQuery(trpc.devices.listEmus.queryOptions());
	const loading = sims.isPending || emus.isPending;
	const sessions: SessionRow[] = [
		...(sims.data ?? []).map((s) => ({
			id: s.id,
			label: s.device,
			kind: "simulator" as const,
			status: s.status,
		})),
		...(emus.data ?? []).map((e) => ({
			id: e.id,
			label: e.profile,
			kind: "emulator" as const,
			status: e.status,
		})),
	];
	const failed = sims.isError || emus.isError;

	// Search box is local for keystroke responsiveness; the URL is the
	// source of truth (debounced). External changes (back/forward, clear)
	// sync back into the box.
	const [draft, setDraft] = useState(q ?? "");
	const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
	useEffect(() => {
		setDraft(q ?? "");
	}, [q]);
	useEffect(
		() => () => {
			if (timer.current) clearTimeout(timer.current);
		},
		[],
	);
	const patchSearch = (patch: Partial<z.infer<typeof searchSchema>>) =>
		navigate({ search: (prev) => ({ ...prev, ...patch }), replace: true });
	const onDraftChange = (value: string) => {
		setDraft(value);
		if (timer.current) clearTimeout(timer.current);
		timer.current = setTimeout(() => {
			patchSearch({ q: value.trim() ? value.trim() : undefined });
		}, SEARCH_DEBOUNCE_MS);
	};

	const kindFiltered = kind
		? sessions.filter((s) => s.kind === kind)
		: sessions;
	const visible = applyTableQuery(
		kindFiltered,
		{ q, sort, dir },
		{
			searchFields: (r) => [r.label, r.id, r.status, r.kind],
			sortValue: (r, column) =>
				column === "kind" ? r.kind : column === "status" ? r.status : r.label,
			defaultSort: "device",
		},
	);
	const filtering = Boolean(q?.trim() || kind);
	const descriptor: SortDescriptor = {
		column: sort ?? "device",
		direction: dir === "desc" ? "descending" : "ascending",
	};
	const onSortChange = (d: SortDescriptor) => {
		const isDefault = d.column === "device" && d.direction === "ascending";
		patchSearch({
			sort: isDefault ? undefined : (d.column as "device" | "kind" | "status"),
			dir: d.direction === "descending" ? "desc" : undefined,
		});
	};
	const clearFilters = () => {
		if (timer.current) clearTimeout(timer.current);
		setDraft("");
		navigate({ search: {}, replace: true });
	};

	return (
		<div className="mx-auto w-full max-w-3xl">
			<p className="font-bold font-jakarta text-nest text-xs uppercase tracking-[0.18em]">
				Devices
			</p>
			<h1 className="mt-3 font-extrabold text-4xl text-white tracking-tight md:text-5xl">
				Simulators<span className="text-nest">.</span>
			</h1>
			<p className="mt-4 max-w-xl text-[15px] text-zinc-400 leading-relaxed">
				{loading
					? "Checking for sessions…"
					: sessions.length > 0
						? `${sessions.length} session${sessions.length === 1 ? "" : "s"} in this org — share links open in any browser, no login needed.`
						: "No sessions yet. Create one below, boot it from the CLI, or let your agent drive — share links open in any browser, no login needed."}
			</p>
			{sessions.length > 0 && (
				<div className="mt-6 space-y-3">
					<div className="flex flex-wrap items-center gap-2">
						<Input
							type="search"
							role="searchbox"
							aria-label="Search sessions"
							placeholder="Search device, id, status…"
							value={draft}
							onChange={(e) => onDraftChange(e.target.value)}
							className="max-w-xs"
						/>
						<fieldset className="flex items-center gap-1">
							<legend className="sr-only">Filter by kind</legend>
							{(
								[
									{ value: undefined, label: "All" },
									{ value: "simulator", label: "Simulators" },
									{ value: "emulator", label: "Emulators" },
								] as const
							).map((opt) => (
								<Button
									key={opt.label}
									size="sm"
									variant={kind === opt.value ? "default" : "ghost"}
									aria-pressed={kind === opt.value}
									onClick={() => patchSearch({ kind: opt.value })}
								>
									{opt.label}
								</Button>
							))}
						</fieldset>
						{filtering && (
							<p className="text-xs text-zinc-500" role="status">
								{visible.length} of {sessions.length} —{" "}
								<button
									type="button"
									onClick={clearFilters}
									className="underline hover:text-white"
								>
									clear
								</button>
							</p>
						)}
					</div>
					<div className="overflow-x-auto rounded-xl border border-white/10 bg-zinc-950">
						<DataTable
							label="Device sessions"
							columns={[
								{
									id: "device",
									header: "Device",
									allowsSorting: true,
									render: (s) => (
										<div className="min-w-0">
											<p className="truncate font-semibold text-sm text-white">
												{s.label}
											</p>
											<p className="truncate font-mono text-xs text-zinc-500">
												{s.id}
											</p>
										</div>
									),
								},
								{
									id: "kind",
									header: "Kind",
									allowsSorting: true,
									render: (s) => (
										<span className="text-xs text-zinc-400">{s.kind}</span>
									),
								},
								{
									id: "status",
									header: "Status",
									allowsSorting: true,
									render: (s) => (
										<span className="inline-block rounded-full border border-nest/30 bg-nest/10 px-2.5 py-1 font-mono text-nest text-xs">
											{s.status}
										</span>
									),
								},
							]}
							rows={visible}
							keyOf={(s) => s.id}
							sortDescriptor={descriptor}
							onSortChange={onSortChange}
							emptyState={
								<div className="space-y-2">
									<p className="text-sm text-zinc-400">
										No sessions match these filters.
									</p>
									<Button size="sm" variant="outline" onClick={clearFilters}>
										Clear search and filters
									</Button>
								</div>
							}
						/>
					</div>
				</div>
			)}
			{failed && (
				<p className="mt-6 text-sm text-zinc-500">
					Couldn't reach the device service — use the CLI commands below
					instead.
				</p>
			)}
			<div className="mt-6">
				<CreateDeviceSplitButton
					caption="Android ready now · iOS coming soon — or paste the commands below."
					checking={loading}
					unavailable={
						failed
							? "Device service is unreachable from this browser session — create from the CLI below instead."
							: null
					}
				/>
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
