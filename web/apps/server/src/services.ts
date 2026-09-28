import { createAuth } from "@sidekick/auth";
import { createDb } from "@sidekick/db";

import { ENV } from "./env.server";

export const db = createDb(ENV);
export const auth = createAuth(ENV, db);
