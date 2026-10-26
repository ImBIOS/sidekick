import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { ConsoleShell } from "@/components/console";
import { authClient } from "@/lib/auth-client";

export const Route = createFileRoute("/_auth")({
	ssr: false,
	component: AuthLayout,
	beforeLoad: async () => {
		const { data: session, error } = await authClient.getSession();
		if (error) {
			// Transient failure (network/429/5xx): fail closed to a retryable
			// error instead of bouncing a possibly-authed user to /login,
			// which would confuse session-expiry triage (see SK-005).
			throw new Error(
				`Couldn't reach the auth service — check your connection and retry. (${error.message ?? error.statusText ?? "fetch failed"})`,
			);
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
