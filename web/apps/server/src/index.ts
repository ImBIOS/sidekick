import { ensureDefaultOrg } from "@sidekick/auth";
import { organization } from "@sidekick/db/schema/organization";
import { trpcServer } from "@hono/trpc-server";
import { appRouter } from "@sidekick/api/routers/index";
import { eq } from "drizzle-orm";
import { Hono } from "hono";
import { cors } from "hono/cors";
import { logger } from "hono/logger";
import { z } from "zod";

import { createContext } from "./context";
import { ENV } from "./env.server";
import { auth } from "./services";
import { db } from "./services";

const app = new Hono();

app.use(logger());
app.use(
  "/*",
  cors({
    origin: ENV.CORS_ORIGIN,
    allowMethods: ["GET", "POST", "OPTIONS"],
    allowHeaders: ["Content-Type", "Authorization"],
    credentials: true,
  }),
);

app.on(["POST", "GET"], "/api/auth/*", async (c) => auth.handler(c.req.raw));

const verifyKeyBody = z.object({ key: z.string().min(1) });

/**
 * Service-to-service key verification for the Go control-plane.
 *
 * The control-plane calls this with its shared secret to resolve an
 * `sk_live_*` agent key to an org. Responses:
 * - 200 `{ valid: true, orgId, userId }`
 * - 401 `{ valid: false }`
 */
app.post("/internal/verify-key", async (c) => {
  const secret = ENV.CONTROL_PLANE_SECRET;
  if (!secret || c.req.header("x-control-plane-secret") !== secret) {
    return c.json({ valid: false, error: "unauthorized" }, 401);
  }
  const parsed = verifyKeyBody.safeParse(await c.req.json().catch(() => ({})));
  if (!parsed.success) {
    return c.json({ valid: false, error: "missing key" }, 400);
  }
  const result = await auth.api.verifyApiKey({ body: { key: parsed.data.key } });
  if (!result.valid || !result.key) {
    return c.json({ valid: false }, 401);
  }
  const referenceId = (result.key as { referenceId: string }).referenceId;

  // Org-scoped keys reference the org directly; user keys reference the user.
  const found = await db
    .select({ id: organization.id })
    .from(organization)
    .where(eq(organization.id, referenceId))
    .limit(1);
  if (found.length > 0 && found[0]) {
    return c.json({ valid: true, orgId: found[0].id, userId: null });
  }
  const orgId = await ensureDefaultOrg(db, { id: referenceId, email: "" });
  return c.json({ valid: true, orgId, userId: referenceId });
});

app.use(
  "/trpc/*",
  trpcServer({
    router: appRouter,
    createContext: (_opts, context) => {
      return createContext({ context });
    },
  }),
);

app.get("/", (c) => {
  return c.text("OK");
});

export default app;
