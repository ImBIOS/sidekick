import { createAuth } from "@openlim-web/auth";
import { createDb } from "@openlim-web/db";

import { ENV } from "./env.server";

export const db = createDb(ENV);
export const auth = createAuth(ENV, db);
