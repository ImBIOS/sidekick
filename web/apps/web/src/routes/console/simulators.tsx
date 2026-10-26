import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/console/simulators")({
	beforeLoad: () => {
		throw redirect({ to: "/simulators" });
	},
});
