import { createFileRoute, Navigate, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { z } from "zod";

import SignInForm from "@/components/sign-in-form";
import SignUpForm from "@/components/sign-up-form";

const searchSchema = z.object({
  mode: z.enum(["signin", "signup"]).optional(),
});

export const Route = createFileRoute("/login")({
  validateSearch: (search) => searchSchema.parse(search),
  head: () => ({
    meta: [{ title: "Log in — Sidekick" }],
  }),
  component: RouteComponent,
});

function RouteComponent() {
  const navigate = useNavigate({ from: "/login" });
  // Legacy ?mode= support: /signup is the canonical sign-up URL now.
  const { mode } = Route.useSearch();
  const [showSignIn, setShowSignIn] = useState(mode !== "signup");

  if (mode === "signup") {
    return <Navigate to="/signup" replace />;
  }

  return showSignIn ? (
    <SignInForm onSwitchToSignUp={() => navigate({ to: "/signup" })} />
  ) : (
    <SignUpForm onSwitchToSignIn={() => setShowSignIn(true)} />
  );
}
