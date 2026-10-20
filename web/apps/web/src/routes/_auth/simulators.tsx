import { Link, createFileRoute } from "@tanstack/react-router";

import { CodeBlock, CreateDeviceSplitButton } from "@/components/console";

export const Route = createFileRoute("/_auth/simulators")({
  component: RouteComponent,
  head: () => ({
    meta: [{ title: "Simulators — Sidekick Console" }],
  }),
});

const COMMANDS = `sidekick ios create --device "iPhone 17"   # or Pixel_7 via: sidekick android create
sidekick ios get <id>                  # poll until running
sidekick ios ax-tree <id>              # read the screen
sidekick android ui-tree <id>          # same verbs, both platforms`;

function RouteComponent() {
  return (
    <div className="mx-auto w-full max-w-3xl">
      <p className="font-jakarta text-xs font-bold tracking-[0.18em] text-nest uppercase">
        Devices
      </p>
      <h1 className="mt-3 text-4xl font-extrabold tracking-tight text-white md:text-5xl">
        Simulators<span className="text-nest">.</span>
      </h1>
      <p className="mt-4 max-w-xl text-[15px] leading-relaxed text-zinc-400">
        No sessions yet. Create one below, boot it from the CLI, or let your
        agent drive — share links open in any browser, no login needed.
      </p>
      <div className="mt-6">
        <CreateDeviceSplitButton caption="iOS or Android. No local setup — or paste the commands below." />
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
