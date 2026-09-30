import { apiKey } from "@better-auth/api-key";
import { drizzleAdapter } from "@better-auth/drizzle-adapter/relations-v2";
import type { Database } from "@sidekick/db";
import { apikey, invitation, member, organization } from "@sidekick/db/schema/organization";
import { eq } from "drizzle-orm";
import { randomUUID } from "node:crypto";
import { betterAuth } from "better-auth";
import { organization as organizationPlugin } from "better-auth/plugins";

import * as schema from "@sidekick/db/schema/auth";
import { user } from "@sidekick/db/schema/auth";

export type AuthConfig = {
  BETTER_AUTH_URL: string;
  BETTER_AUTH_SECRET: string;
  CORS_ORIGIN: string;
};

export const OWNER_ROLE = "owner";

/** Build a URL-safe slug base from an email address. Uniqueness comes from the random suffix. */
export function slugBaseForEmail(email: string): string {
  const local = email.split("@")[0] ?? "org";
  const slug = local
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 32);
  return slug || "org";
}

/** Display name for a fresh default org, e.g. "ada" -> "Ada's Org". */
export function defaultOrgName(userName: string | null | undefined, email: string): string {
  const first = userName?.trim().split(/\s+/)[0] || email.split("@")[0] || "Personal";
  const capitalized = first.charAt(0).toUpperCase() + first.slice(1);
  return `${capitalized}'s Org`;
}

/**
 * Ensure the user belongs to at least one organization, creating a default
 * org (owned by the user) when they have none. Idempotent: safe to call from
 * both the user-create hook and the session-create hook (backfill).
 *
 * Every agent path (remote builds, sims, emus) is org-scoped, so this is the
 * invariant that makes `sidekick login` work out of the box with zero setup.
 *
 * @returns the default (earliest) organization id for the user.
 */
export async function ensureDefaultOrg(
  database: Database,
  user: { id: string; name?: string | null; email: string },
): Promise<string> {
  const existing = await database
    .select({ organizationId: member.organizationId })
    .from(member)
    .where(eq(member.userId, user.id))
    .limit(1);
  if (existing.length > 0 && existing[0]) {
    return existing[0].organizationId;
  }
  const now = new Date();
  const orgId = randomUUID();
  await database.insert(organization).values({
    id: orgId,
    name: defaultOrgName(user.name, user.email),
    slug: `${slugBaseForEmail(user.email)}-${randomUUID().slice(0, 8)}`,
    createdAt: now,
  });
  await database.insert(member).values({
    id: randomUUID(),
    organizationId: orgId,
    userId: user.id,
    role: OWNER_ROLE,
    createdAt: now,
  });
  return orgId;
}

export function createAuth(
  env: AuthConfig,
  database: Database,
  desktopOrigins: readonly string[] = [],
) {
  return betterAuth({
    database: drizzleAdapter(database, {
      provider: "pg",
      schema: {
        ...schema,
        organization,
        member,
        invitation,
        apikey,
      },
    }),
    trustedOrigins: [env.CORS_ORIGIN, ...desktopOrigins],
    emailAndPassword: { enabled: true },
    secret: env.BETTER_AUTH_SECRET,
    baseURL: env.BETTER_AUTH_URL,
    advanced: {
      defaultCookieAttributes: {
        sameSite: "none",
        secure: true,
        httpOnly: true,
      },
    },
    databaseHooks: {
      user: {
        create: {
          // Provision the default org at signup so agents/CLI work immediately.
          after: async (user) => {
            await ensureDefaultOrg(database, {
              id: user.id,
              name: user.name,
              email: user.email,
            });
          },
        },
      },
      session: {
        create: {
          // Pin the session to the default org (backfills if signup hook raced).
          // NOTE: this hook fires before user.create.after on signup, so load
          // the user row for real name/email instead of a junk fallback name.
          before: async (session) => {
            const rows = await database
              .select({ id: user.id, name: user.name, email: user.email })
              .from(user)
              .where(eq(user.id, session.userId))
              .limit(1);
            const owner =
              rows.length > 0 && rows[0]
                ? { id: rows[0].id, name: rows[0].name, email: rows[0].email }
                : { id: session.userId, email: "" };
            const orgId = await ensureDefaultOrg(database, owner);
            return { data: { ...session, activeOrganizationId: orgId } };
          },
        },
      },
    },
    plugins: [
      organizationPlugin({
        // Agents invite teammates via console; no mailer wired yet.
        sendInvitationEmail: async () => {},
      }),
      // Org-owned `sk_live_*` keys: this is what `sidekick login` stores and
      // the control-plane verifies for remote builds/sims/emus.
      // `references: "organization"` stores the org id as the key owner, so
      // listing/verifying is scoped by org, not by user.
      apiKey({
        references: "organization",
        defaultPrefix: "sk_live_",
        rateLimit: { enabled: false },
      }),
    ],
  });
}

export type Session = ReturnType<typeof createAuth>["$Infer"]["Session"];
