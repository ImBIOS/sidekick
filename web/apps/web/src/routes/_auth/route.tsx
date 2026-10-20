import { Outlet, createFileRoute, redirect } from "@tanstack/react-router";

import { authClient } from "@/lib/auth-client";
import { ConsoleShell } from "@/components/console";

export const Route = createFileRoute("/_auth")({
  ssr: false,
  component: AuthLayout,
  beforeLoad: async () => {
    let session;
    try {
      const res = await authClient.getSession();
      session = res.data;
    } catch {
      session = null;
    }
    if (!session) {
      throw redirect({
        to: "/login",
      });
    }
    return { session };
  },
});

function AuthLayout() {
  return (
    <ConsoleShell>
      <Outlet />
    </ConsoleShell>
  );
}
