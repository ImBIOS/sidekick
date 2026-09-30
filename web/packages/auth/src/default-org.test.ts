import { PGlite } from "@electric-sql/pglite";
import type { Database } from "@sidekick/db";
import { apikey, member, organization } from "@sidekick/db/schema/organization";
import { eq } from "drizzle-orm";
import { drizzle } from "drizzle-orm/pglite";
import { describe, expect, test } from "bun:test";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

import { createAuth, defaultOrgName, ensureDefaultOrg, slugBaseForEmail } from "./index";

const here = dirname(fileURLToPath(import.meta.url));
const migrationsDir = join(here, "..", "..", "db", "src", "migrations");

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

/** Session cookie headers, the same shape the tRPC context forwards. */
async function authedHeaders(
  auth: ReturnType<typeof createAuth>,
  email: string,
  password: string,
): Promise<Headers> {
  const res = (await auth.api.signInEmail({
    body: { email, password },
    asResponse: true,
  } as Parameters<typeof auth.api.signInEmail>[0])) as unknown as Response;
  const setCookie = res.headers.get("set-cookie") ?? "";
  const cookie = setCookie.split(";")[0] ?? "";
  return new Headers({ cookie });
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

describe("default org provisioning", () => {
  test("slug + name helpers", () => {
    expect(slugBaseForEmail("Ada.Lovelace+1@example.com")).toBe("ada-lovelace-1");
    expect(slugBaseForEmail("x@y.z")).toBe("x");
    expect(defaultOrgName("Ada Lovelace", "ada@example.com")).toBe("Ada's Org");
    expect(defaultOrgName(null, "bob@example.com")).toBe("Bob's Org");
  });

  test("signup creates an owned default org and pins the session to it", async () => {
    const db = await testDb();
    const auth = testAuth(db);

    await auth.api.signUpEmail({
      body: { name: "Ada Lovelace", email: "ada@example.com", password: "password123" },
    });

    const orgs = await db.select().from(organization);
    expect(orgs.length).toBe(1);
    expect(orgs[0]?.name).toBe("Ada's Org");
    expect(orgs[0]?.slug.startsWith("ada-")).toBe(true);

    const members = await db.select().from(member);
    expect(members.length).toBe(1);
    expect(members[0]?.role).toBe("owner");
    expect(members[0]?.organizationId).toBe(orgs[0]?.id);

    const authed = await authedHeaders(auth, "ada@example.com", "password123");
    const gotSession = await auth.api.getSession({ headers: authed });
    const defaultOrgId = orgs[0]?.id;
    expect(defaultOrgId).toBeTruthy();
    expect(gotSession?.session.activeOrganizationId).toBe(defaultOrgId as string);

    // Idempotent: backfill returns the same org, no duplicates.
    const again = await ensureDefaultOrg(db as unknown as Database, {
      id: members[0]?.userId ?? "",
      email: "ada@example.com",
    });
    expect(again).toBe(defaultOrgId as string);
    expect((await db.select().from(organization)).length).toBe(1);
  });

  test("org-scoped sk_live_ key verifies and resolves to the org", async () => {
    const db = await testDb();
    const auth = testAuth(db);

    await auth.api.signUpEmail({
      body: { name: "Grace", email: "grace@example.com", password: "password123" },
    });
    const orgs = await db.select().from(organization);
    const orgId = orgs[0]?.id ?? "";

    const headers = await authedHeaders(auth, "grace@example.com", "password123");

    const created = (await auth.api.createApiKey({
      body: { name: "sidekick CLI", organizationId: orgId },
      headers,
    })) as unknown as { key: string; id: string };
    expect(created.key.startsWith("sk_live_")).toBe(true);

    const verified = await auth.api.verifyApiKey({ body: { key: created.key } });
    expect(verified.valid).toBe(true);

    // Same resolution the /internal/verify-key endpoint performs.
    const referenceId = (verified.key as unknown as { referenceId: string }).referenceId;
    const found = await db
      .select({ id: organization.id })
      .from(organization)
      .where(eq(organization.id, referenceId))
      .limit(1);
    const resolvedOrgId =
      found.length > 0
        ? found[0]?.id
        : await ensureDefaultOrg(db as unknown as Database, { id: referenceId, email: "" });
    expect(resolvedOrgId).toBe(orgId);

    const rows = await db.select().from(apikey);
    expect(rows.length).toBe(1);
    expect(rows[0]?.id).toBe(created.id);

    const listed = (await auth.api.listApiKeys({
      headers,
      query: { organizationId: orgId },
    })) as unknown as { apiKeys: { id: string }[] };
    expect(listed.apiKeys.map((k) => k.id)).toContain(created.id);
  });
});
