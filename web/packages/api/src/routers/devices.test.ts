import { PGlite } from "@electric-sql/pglite";
import { createAuth } from "@sidekick/auth";
import type { Database } from "@sidekick/db";
import { TRPCError } from "@trpc/server";
import { describe, expect, test } from "bun:test";
import { drizzle } from "drizzle-orm/pglite";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { appRouter } from "./index";

const here = dirname(fileURLToPath(import.meta.url));
const migrationsDir = join(here, "..", "..", "..", "db", "src", "migrations");

async function testDb() {
  const client = new PGlite();
  const folders = readdirSync(migrationsDir)
    .filter((f) => statSync(join(migrationsDir, f)).isDirectory())
    .sort();
  for (const folder of folders) {
    const sql = readFileSync(join(migrationsDir, folder, "migration.sql"), "utf8");
    for (const stmt of sql.split("--> statement-breakpoint")) {
      const trimmed = stmt.trim();
      if (trimmed) {
        await client.exec(trimmed);
      }
    }
  }
  return drizzle({ client });
}

function testAuth(db: ReturnType<typeof drizzle>) {
  return createAuth(
    {
      BETTER_AUTH_URL: "http://localhost:8000",
      BETTER_AUTH_SECRET: "test-secret-that-is-at-least-32-chars",
      CORS_ORIGIN: "http://localhost:3000",
    },
    db as unknown as Database,
  );
}

async function authedCaller(auth: ReturnType<typeof createAuth>, db: ReturnType<typeof drizzle>) {
  await auth.api.signUpEmail({
    body: { name: "Test 1", email: "test01@sidekick.imbios.dev", password: "password123" },
  });
  const res = (await auth.api.signInEmail({
    body: { email: "test01@sidekick.imbios.dev", password: "password123" },
    asResponse: true,
  } as Parameters<typeof auth.api.signInEmail>[0])) as unknown as Response;
  const headers = new Headers({ cookie: (res.headers.get("set-cookie") ?? "").split(";")[0] ?? "" });
  const session = await auth.api.getSession({ headers });
  return {
    auth,
    headers,
    caller: appRouter.createCaller({
      session,
      db: db as unknown as Database,
      headers,
      auth,
    }),
  };
}

/** Minimal control-plane double: Bearer-gated sim/emu store, 404s when missing. */
function stubControlPlane() {
  const seenSecrets = new Set<string>();
  const sims = new Map<string, Record<string, unknown>>();
  const emus = new Map<string, Record<string, unknown>>();
  let n = 0;

  const server = Bun.serve({
    port: 0,
    async fetch(req) {
      const secret = (req.headers.get("authorization") ?? "").replace(/^Bearer /, "");
      if (!secret) return new Response("unauthorized", { status: 401 });
      seenSecrets.add(secret);
      const url = new URL(req.url);
      const [, , kind, id, sub] = url.pathname.split("/");
      if (kind !== "sims" && kind !== "emus") return new Response("not found", { status: 404 });
      const store = kind === "sims" ? sims : emus;
      if (req.method === "POST" && !id) {
        const body = (await req.json().catch(() => ({}))) as Record<string, string>;
        n += 1;
        if (kind === "sims" && !body.device) {
          return Response.json({ error: "device is required" }, { status: 400 });
        }
        if (kind === "emus" && !body.profile) {
          return Response.json({ error: "profile is required" }, { status: 400 });
        }
        const record = {
          id: `${kind === "sims" ? "sim" : "emu"}_test-${n}`,
          status: "provisioning",
          ...(kind === "sims" ? { device: body.device } : { profile: body.profile }),
          ...(body.pool ? { pool: body.pool } : {}),
          ...(body.region ? { region: body.region } : {}),
        };
        store.set(record.id as string, record);
        return Response.json(record, { status: 201 });
      }
      if (req.method === "GET" && !id) {
        return Response.json({ [kind]: [...store.values()] });
      }
      if (req.method === "GET" && id && !sub) {
        const record = store.get(id);
        if (!record) {
          return Response.json({ error: `${kind === "sims" ? "sim" : "emu"} not found` }, { status: 404 });
        }
        return Response.json(record);
      }
      return new Response("not found", { status: 404 });
    },
  });

  return {
    url: `http://localhost:${server.port}`,
    seenSecrets,
    stop: () => server.stop(),
  };
}

describe("devices router (click-first provisioning)", () => {
  test("empty lists and missing ids resolve to [] and null", async () => {
    const db = await testDb();
    const auth = testAuth(db);
    const { caller } = await authedCaller(auth, db);
    const stub = stubControlPlane();
    const prev = process.env.SIDEKICK_CONTROL_URL;
    process.env.SIDEKICK_CONTROL_URL = stub.url;
    try {
      expect(await caller.devices.listSims()).toEqual([]);
      expect(await caller.devices.listEmus()).toEqual([]);
      expect(await caller.devices.getSim({ id: "nope" })).toBeNull();
      expect(await caller.devices.getEmu({ id: "nope" })).toBeNull();
      for (const secret of stub.seenSecrets) {
        expect(secret.startsWith("sk_live_")).toBe(true);
      }
    } finally {
      if (prev === undefined) delete process.env.SIDEKICK_CONTROL_URL;
      else process.env.SIDEKICK_CONTROL_URL = prev;
      stub.stop();
    }
  });

  test("create sim + emu appear in lists, ephemeral key is revoked", async () => {
    const db = await testDb();
    const auth = testAuth(db);
    const { caller, headers } = await authedCaller(auth, db);
    const stub = stubControlPlane();
    const prev = process.env.SIDEKICK_CONTROL_URL;
    process.env.SIDEKICK_CONTROL_URL = stub.url;
    try {
      const sim = await caller.devices.createSim({ pool: "emu-sg-1" });
      expect(sim.device).toBe("iPhone 17");
      expect((sim as { pool?: string }).pool).toBe("emu-sg-1");
      const emu = await caller.devices.createEmu({});
      expect(emu.profile).toBe("Pixel_7");

      expect((await caller.devices.listSims()).length).toBe(1);
      expect((await caller.devices.listEmus()).length).toBe(1);
      expect(await caller.devices.getSim({ id: sim.id })).toMatchObject({ id: sim.id });

      const keys = (await auth.api.listApiKeys({ headers })) as {
        apiKeys: unknown[];
      };
      expect(keys.apiKeys.length).toBe(0);
    } finally {
      if (prev === undefined) delete process.env.SIDEKICK_CONTROL_URL;
      else process.env.SIDEKICK_CONTROL_URL = prev;
      stub.stop();
    }
  });

  test("unreachable control-plane maps to PRECONDITION_FAILED", async () => {
    const db = await testDb();
    const auth = testAuth(db);
    const { caller } = await authedCaller(auth, db);
    const prev = process.env.SIDEKICK_CONTROL_URL;
    process.env.SIDEKICK_CONTROL_URL = "http://localhost:1";
    try {
      const err = await caller.devices.createSim({}).then(
        () => null,
        (e: unknown) => e,
      );
      expect(err).toBeInstanceOf(TRPCError);
      expect((err as TRPCError).code).toBe("PRECONDITION_FAILED");
    } finally {
      if (prev === undefined) delete process.env.SIDEKICK_CONTROL_URL;
      else process.env.SIDEKICK_CONTROL_URL = prev;
    }
  });
});
