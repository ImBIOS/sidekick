import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/console/api-keys")({
	beforeLoad: () => {
		throw redirect({ to: "/settings" });
	},
});
