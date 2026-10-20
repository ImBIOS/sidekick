import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { toast } from "sonner";

import { Button } from "@sidekick/ui/components/button";
import { Input } from "@sidekick/ui/components/input";
import { Label } from "@sidekick/ui/components/label";
import { CodeBlock, CopyButton, Step } from "@/components/console";
import { useTRPC } from "@/utils/trpc";

export const Route = createFileRoute("/_auth/console")({
  component: RouteComponent,
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
  const greeting = hour < 12 ? "Good morning" : hour < 18 ? "Good afternoon" : "Good evening";

  const trpc = useTRPC();
  const queryClient = useQueryClient();
  const [newSecret, setNewSecret] = useState<string | null>(null);
  const keys = useQuery(trpc.apiKeys.list.queryOptions());
  const hasKey = (keys.data?.length ?? 0) > 0;

  const createKey = useMutation(
    trpc.apiKeys.create.mutationOptions({
      onSuccess: (data) => {
        setNewSecret(data.key);
        toast.success("API key created — copy it now, it won't be shown again");
        queryClient.invalidateQueries({ queryKey: trpc.apiKeys.list.queryKey() });
      },
      onError: (e) => toast.error(e.message),
    }),
  );

  return (
    <div className="mx-auto w-full max-w-3xl">
      <p className="font-jakarta text-xs font-bold tracking-[0.18em] text-nest uppercase">
        Your first run
      </p>
      <h1 className="mt-3 text-4xl font-extrabold tracking-tight text-white md:text-5xl">
        From code to a live device<span className="text-nest">.</span>
      </h1>
      <p className="mt-4 max-w-xl text-[15px] leading-relaxed text-zinc-400">
        {greeting}, {session.user.name}. Start a cloud device, give your coding
        agent a prompt, and watch your app come to life. Three steps, about two
        minutes — no Mac, no Xcode, no installs beyond the CLI.
      </p>

      <div className="mt-10 space-y-10">
        <Step n={1} title="Copy your API key">
          <p>
            Keys are scoped to your default org. The secret shows{" "}
            <strong className="text-zinc-200">once</strong> — agents pick it up
            from <code className="font-mono text-xs">SIDEKICK_TOKEN</code>.
          </p>
          {newSecret ? (
            <div className="space-y-2 rounded-xl border border-dashed border-nest/40 bg-nest/5 p-3">
              <Label>New key (shown once)</Label>
              <div className="flex gap-2">
                <Input readOnly value={newSecret} className="font-mono text-xs" />
                <CopyButton text={newSecret} />
                <Button variant="ghost" size="sm" onClick={() => setNewSecret(null)}>
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
                  You have {keys.data?.length} key{keys.data?.length === 1 ? "" : "s"} — manage
                  them in <Link to="/settings" className="underline hover:text-white">API keys</Link>.
                </p>
              )}
            </div>
          )}
        </Step>

        <Step n={2} title="Install the CLI and log in">
          <p>
            One binary, no sandbox changes. <code className="font-mono text-xs">agent-setup</code>{" "}
            registers the skill and MCP server so Claude Code, Cursor, or Codex can drive devices.
          </p>
          <CodeBlock lines={INSTALL} title="terminal" />
        </Step>

        <Step n={3} title="Build it, boot it, tap it">
          <p>
            Remote Xcode compiles from source, a simulator boots in the cloud, and every verb
            returns something your agent can read: trees, screenshots, videos, logs.
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

      <div className="mt-12 flex flex-wrap items-center justify-between gap-3 border-t border-white/10 pt-6 text-sm text-zinc-500">
        <p>Bring your app and the agent you already use.</p>
        <div className="flex gap-4">
          <Link to="/simulators" className="underline hover:text-white">
            Simulators
          </Link>
          <Link to="/builds" className="underline hover:text-white">
            Builds
          </Link>
          <a href="https://github.com/ImBIOS/sidekick" className="underline hover:text-white">
            Support ↗
          </a>
        </div>
      </div>
    </div>
  );
}
