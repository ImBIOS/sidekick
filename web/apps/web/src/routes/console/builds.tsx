import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/console/builds")({
	beforeLoad: () => {
		throw redirect({ to: "/builds" });
	},
});
