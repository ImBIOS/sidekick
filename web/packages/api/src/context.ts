import type { Session } from "@sidekick/auth";
import type { Database } from "@sidekick/db";

export type Context = {
  session: Session | null;
  db: Database;
};
