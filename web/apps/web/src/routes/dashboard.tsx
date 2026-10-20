import { createFileRoute, redirect } from "@tanstack/react-router";

// The console lived at /dashboard during the first week. Keep old
// bookmarks (and muscle memory) working.
export const Route = createFileRoute("/dashboard")({
  beforeLoad: () => {
    throw redirect({ to: "/console" });
  },
});
