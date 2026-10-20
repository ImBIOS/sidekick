import { Link, createFileRoute } from "@tanstack/react-router";

import { CodeBlock } from "@/components/console";

export const Route = createFileRoute("/_auth/simulators")({
  component: RouteComponent,
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
        No sessions yet. Boot one from the CLI or your agent — it appears here
        with its share link the moment it is running.
      </p>
      <div className="mt-8">
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
