import type { Session, createAuth } from "@sidekick/auth";
import type { Database } from "@sidekick/db";

export type Auth = ReturnType<typeof createAuth>;

export type Context = {
  session: Session | null;
  db: Database;
  /** Raw request headers: forwarded to better-auth server APIs (session-bound). */
  headers: Headers;
  auth: Auth;
};
