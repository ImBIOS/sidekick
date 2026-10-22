import { createFileRoute, useNavigate } from "@tanstack/react-router";

import SignUpForm from "@/components/sign-up-form";

export const Route = createFileRoute("/signup")({
  head: () => ({
    meta: [{ title: "Sign up — Sidekick" }],
  }),
  component: RouteComponent,
});

function RouteComponent() {
  const navigate = useNavigate();
  return <SignUpForm onSwitchToSignIn={() => navigate({ to: "/login", search: { mode: "signin" } })} />;
}
