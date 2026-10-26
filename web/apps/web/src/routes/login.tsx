import { createFileRoute, isRedirect, redirect } from "@tanstack/react-router";
import { useState } from "react";
import { z } from "zod";

import SignInForm from "@/components/sign-in-form";
import SignUpForm from "@/components/sign-up-form";

const searchSchema = z.object({
	mode: z.enum(["signin", "signup"]).optional(),
	plan: z.string().max(32).optional(),
});

export const Route = createFileRoute("/login")({
	validateSearch: (search) => searchSchema.parse(search),
	head: () => ({
		meta: [{ title: "Log in — Sidekick" }],
	}),
	beforeLoad: async () => {
		try {
			const { authClient } = await import("@/lib/auth-client");
			const { data: session } = await authClient.getSession();
			if (session) {
				throw redirect({ to: "/console" });
			}
		} catch (e) {
			if (isRedirect(e)) throw e;
			// Not signed in (or auth unreachable) — stay on /login.
		}
		return {};
	},
	component: RouteComponent,
});

function RouteComponent() {
	const { mode, plan } = Route.useSearch();
	const [showSignIn, setShowSignIn] = useState(mode === "signin");

	return (
		<>
			{plan && (
				<p
					className="mx-auto mt-6 max-w-md rounded-xl border border-nest/30 bg-nest/10 px-4 py-3 text-center text-sm text-zinc-200"
					role="status"
				>
					You picked <strong className="font-semibold">{plan}</strong> — create
					your account to get started.
				</p>
			)}
			{showSignIn ? (
				<SignInForm onSwitchToSignUp={() => setShowSignIn(false)} />
			) : (
				<SignUpForm onSwitchToSignIn={() => setShowSignIn(true)} />
			)}
		</>
	);
}
