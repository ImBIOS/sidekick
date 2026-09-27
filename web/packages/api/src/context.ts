import type { Session } from "@openlim-web/auth";
import type { Database } from "@openlim-web/db";

export type Context = {
  session: Session | null;
  db: Database;
};
