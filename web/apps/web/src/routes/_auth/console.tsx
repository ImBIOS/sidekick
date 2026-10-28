import { Button } from "@sidekick/ui/components/button";
import { Input } from "@sidekick/ui/components/input";
import { Label } from "@sidekick/ui/components/label";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";
import {
	CodeBlock,
	CopyButton,
	CreateDeviceSplitButton,
	Step,
} from "@/components/console";
import { useTRPC } from "@/utils/trpc";

export const Route = createFileRoute("/_auth/console")({
	component: RouteComponent,
	head: () => ({
		meta: [{ title: "Get started — Sidekick Console" }],
	}),
});

const INSTALL = `npm i -g @imbios/sidekick   # or: brew install ImBIOS/tap/sidekick
sidekick login                # paste the key from step 1
sidekick agent-setup          # skill + MCP for Claude Code, Cursor, Codex`;

const FIRST_RUN = `sidekick xcode build . --scheme MyApp
sidekick ios create --device "iPhone 17"
sidekick ios ax-tree <id>     # read the screen first, always
sidekick ios act <id> --tap @e7
sidekick ios screenshot <id>  # looks right? ship it`;

function RouteComponent() {
	const { session } = Route.useRouteContext();
	const hour = new Date().getHours();
	const greeting =
		hour < 12 ? "Good morning" : hour < 18 ? "Good afternoon" : "Good evening";

	const trpc = useTRPC();
	const queryClient = useQueryClient();
	const [newSecret, setNewSecret] = useState<string | null>(null);
	const emusProbe = useQuery({
		...trpc.devices.listEmus.queryOptions(),
		staleTime: 60_000,
		retry: false,
	});
	const fleetProbe = useQuery({
		...trpc.devices.fleetRegions.queryOptions(),
		staleTime: 30_000,
		retry: false,
	});
	const liveRegions = fleetProbe.data ?? [];
	const fleetLabel = fleetProbe.isPending
		? "Checking fleet…"
		: fleetProbe.isError
			? "Fleet status unavailable"
			: liveRegions.length === 0
				? "No live regions — no hosts polling yet"
				: `Live: ${liveRegions.map((r) => `${r.region} (${r.hosts} host${r.hosts === 1 ? "" : "s"}, ${r.queued} queued)`).join(" · ")}`;
	const keys = useQuery(trpc.apiKeys.list.queryOptions());
	const activeOrg = useQuery(trpc.organization.active.queryOptions());
	const hasKey = (keys.data?.length ?? 0) > 0;

	const createKey = useMutation(
		trpc.apiKeys.create.mutationOptions({
			onSuccess: (data) => {
				setNewSecret(data.key);
				toast.success("API key created — copy it now, it won't be shown again");
				queryClient.invalidateQueries({
					queryKey: trpc.apiKeys.list.queryKey(),
				});
			},
			onError: (e) => toast.error(e.message),
		}),
	);

	return (
		<div className="mx-auto w-full max-w-3xl">
			<p className="font-bold font-jakarta text-nest text-xs uppercase tracking-[0.18em]">
				Your first run
			</p>
			<h1 className="mt-3 font-extrabold text-4xl text-white tracking-tight md:text-5xl">
				From code to a live device<span className="text-nest">.</span>
			</h1>
			<p className="mt-4 max-w-xl text-[15px] text-zinc-400 leading-relaxed">
				{greeting}, {session.user.name}. Start a cloud device, give your coding
				agent a prompt, and watch your app come to life. Three steps, about two
				minutes — no Mac, no Xcode, no installs beyond the CLI.
			</p>

			<div className="mt-6">
				<CreateDeviceSplitButton
					caption="Android ready now · iOS coming soon."
					checking={emusProbe.isPending}
					unavailable={
						emusProbe.isError
							? "Device service is unreachable from this browser session — use the CLI commands below instead."
							: null
					}
				/>
			</div>

			<div className="mt-8 grid gap-3 sm:grid-cols-3">
				<div className="rounded-2xl border border-white/10 bg-zinc-950 p-5">
					<p className="font-semibold text-xs text-zinc-500 uppercase tracking-[0.14em]">
						Fleet regions
					</p>
					<p className="mt-1 font-semibold text-lg text-white">
						{fleetProbe.data ? liveRegions.length : "…"}
					</p>
					<p className="text-xs text-zinc-500" role="status">
						{fleetLabel}
					</p>
					<p className="mt-1 font-mono text-[11px] text-zinc-600">
						sidekick fleet regions
					</p>
				</div>
				<div className="rounded-2xl border border-white/10 bg-zinc-950 p-5">
					<p className="font-semibold text-xs text-zinc-500 uppercase tracking-[0.14em]">
						Organization
					</p>
					<p className="mt-1 truncate font-semibold text-lg text-white">
						{activeOrg.data?.name ?? "…"}
					</p>
					<p className="truncate font-mono text-xs text-zinc-500">
						{activeOrg.data?.slug ?? ""}
					</p>
				</div>
				<div className="rounded-2xl border border-white/10 bg-zinc-950 p-5">
					<p className="font-semibold text-xs text-zinc-500 uppercase tracking-[0.14em]">
						API keys
					</p>
					<p className="mt-1 font-semibold text-lg text-white">
						{keys.data ? keys.data.length : "…"}
					</p>
					<p className="text-xs text-zinc-500">
						{hasKey ? (
							<Link to="/settings" className="underline hover:text-white">
								Manage in API keys
							</Link>
						) : (
							"Create your first below"
						)}
					</p>
				</div>
			</div>

			<div className="mt-10 space-y-10">
				<Step n={1} title="Copy your API key">
					<p>
						Keys are scoped to your default org. The secret shows{" "}
						<strong className="text-zinc-200">once</strong> — agents pick it up
						from <code className="font-mono text-xs">SIDEKICK_TOKEN</code>.
					</p>
					{newSecret ? (
						<div className="space-y-2 rounded-xl border border-nest/40 border-dashed bg-nest/5 p-3">
							<Label>New key (shown once)</Label>
							<div className="flex gap-2">
								<Input
									readOnly
									value={newSecret}
									className="font-mono text-xs"
								/>
								<CopyButton text={newSecret} />
								<Button
									variant="ghost"
									size="sm"
									onClick={() => setNewSecret(null)}
								>
									Dismiss
								</Button>
							</div>
						</div>
					) : (
						<div>
							<Button
								disabled={createKey.isPending}
								onClick={() => createKey.mutate({ name: "sidekick CLI" })}
								className="rounded-full bg-white px-5 text-black hover:bg-zinc-200"
							>
								{hasKey ? "Create another key" : "Create API key"}
							</Button>
							{hasKey && (
								<p className="mt-2 text-xs">
									You have {keys.data?.length} key
									{keys.data?.length === 1 ? "" : "s"} — manage them in{" "}
									<Link to="/settings" className="underline hover:text-white">
										API keys
									</Link>
									.
								</p>
							)}
						</div>
					)}
				</Step>

				<Step n={2} title="Install the CLI and log in">
					<p>
						One binary, no sandbox changes.{" "}
						<code className="font-mono text-xs">agent-setup</code> registers the
						skill and MCP server so Claude Code, Cursor, or Codex can drive
						devices.
					</p>
					<CodeBlock lines={INSTALL} title="terminal" />
				</Step>

				<Step n={3} title="Build it, boot it, tap it">
					<p>
						Remote Xcode compiles from source, a simulator boots in the cloud,
						and every verb returns something your agent can read: trees,
						screenshots, videos, logs.
					</p>
					<CodeBlock lines={FIRST_RUN} title="terminal" />
					<p>
						Full verb list lives in the{" "}
						<a href="/llms.txt" className="underline hover:text-white">
							skill file
						</a>{" "}
						— or compare us first:{" "}
						<Link to="/alternatives" className="underline hover:text-white">
							Sidekick vs the 9 alternatives
						</Link>
						.
					</p>
				</Step>
			</div>

			<div className="mt-12 flex flex-wrap items-center justify-between gap-3 border-white/10 border-t pt-6 text-sm text-zinc-500">
				<p>Bring your app and the agent you already use.</p>
				<div className="flex gap-4">
					<Link to="/simulators" className="underline hover:text-white">
						Simulators
					</Link>
					<Link to="/builds" className="underline hover:text-white">
						Builds
					</Link>
					<a
						href="https://github.com/ImBIOS/sidekick"
						className="underline hover:text-white"
					>
						Support ↗
					</a>
				</div>
			</div>
		</div>
	);
}
