import { Link, createFileRoute } from "@tanstack/react-router";

import { CodeBlock } from "@/components/console";

export const Route = createFileRoute("/_auth/builds")({
  component: RouteComponent,
});

const COMMANDS = `sidekick xcode build . --scheme MyApp   # streams logs until done
sidekick xcode build . --scheme MyApp --wait=false   # queue and walk away`;

function RouteComponent() {
  return (
    <div className="mx-auto w-full max-w-3xl">
      <p className="font-jakarta text-xs font-bold tracking-[0.18em] text-nest uppercase">
        Builds
      </p>
      <h1 className="mt-3 text-4xl font-extrabold tracking-tight text-white md:text-5xl">
        Builds<span className="text-nest">.</span>
      </h1>
      <p className="mt-4 max-w-xl text-[15px] leading-relaxed text-zinc-400">
        No builds yet. Queue one from the CLI or your agent — remote Xcode and
        Gradle compile from source with a warm cache, and logs stream back live.
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
