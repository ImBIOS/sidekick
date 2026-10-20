import { TRPCError } from "@trpc/server";
import { z } from "zod";

import type { Auth } from "../context";

import { protectedProcedure, router } from "../index";

/**
 * Click-first device provisioning for the console.
 *
 * Proxies the Go control-plane (`POST/GET /v1/sims`, `POST/GET /v1/emus`)
 * using an ephemeral org-scoped key: mint via better-auth, call, revoke.
 * Zero Go changes; per-org isolation comes from the existing
 * `/internal/verify-key` flow. Missing resources resolve to `null`/`[]`
 * (never 404s to the UI) so empty states can offer creation.
 */

const simSchema = z.looseObject({
  id: z.string(),
  device: z.string(),
  status: z.string(),
});

const emuSchema = z.looseObject({
  id: z.string(),
  profile: z.string(),
  status: z.string(),
});

function controlBase(): string {
  const raw =
    process.env.SIDEKICK_CONTROL_URL ?? process.env.SIDEKICK_CONTROL ?? "http://localhost:8080";
  return raw.replace(/\/+$/, "");
}

async function controlFetch(secret: string, path: string, init?: RequestInit) {
  let res: Response;
  try {
    res = await fetch(`${controlBase()}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${secret}`,
        ...(init?.headers ?? {}),
      },
    });
  } catch {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message:
        "Control-plane unreachable — create from the CLI instead: sidekick ios create --device \"iPhone 17\"",
    });
  }
  if (res.status === 404) return null;
  const body = (await res.json().catch(() => ({}))) as {
    error?: string;
  };
  if (!res.ok) {
    throw new TRPCError({
      code: "BAD_REQUEST",
      message: body.error ?? `Control-plane rejected the request (${res.status})`,
    });
  }
  return body;
}

/** Mint an ephemeral key, run `fn`, revoke. Revocation failure is swallowed.
 * TTL is 1 day (the plugin minimum) as a backstop; revocation is the real
 * cleanup, so a leaked key visibly named `console one-click` expires fast. */
async function withEphemeralKey<T>(
  ctx: { auth: Auth; headers: Headers },
  orgId: string | null | undefined,
  fn: (secret: string) => Promise<T>,
): Promise<T> {
  const created = (await ctx.auth.api.createApiKey({
    body: { name: "console one-click", organizationId: orgId ?? undefined, expiresIn: 86400 },
    headers: ctx.headers,
  })) as { key: string; id: string };
  try {
    return await fn(created.key);
  } finally {
    await ctx.auth.api
      .deleteApiKey({ body: { keyId: created.id }, headers: ctx.headers })
      .catch(() => undefined);
  }
}

function orgIdOf(ctx: { session: unknown }): string | null | undefined {
  const s = ctx.session as {
    session: { activeOrganizationId?: unknown };
  } | null;
  return s?.session.activeOrganizationId as string | null | undefined;
}

export const devicesRouter = router({
  listSims: protectedProcedure.query(async ({ ctx }) => {
    const orgId = orgIdOf(ctx);
    const body = await withEphemeralKey(ctx, orgId, (secret) => controlFetch(secret, "/v1/sims"));
    const sims = ((body as { sims?: unknown } | null)?.sims ?? []) as unknown[];
    return simSchema.array().parse(sims);
  }),

  getSim: protectedProcedure
    .input(z.object({ id: z.string().min(1) }))
    .query(async ({ ctx, input }) => {
      const body = await withEphemeralKey(ctx, orgIdOf(ctx), (secret) =>
        controlFetch(secret, `/v1/sims/${encodeURIComponent(input.id)}`),
      );
      if (body === null) return null;
      return simSchema.parse(body);
    }),

  createSim: protectedProcedure
    .input(
      z.object({
        device: z.string().min(1).default("iPhone 17"),
        runtime: z.string().max(64).nullish(),
        region: z.string().max(64).nullish(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const body = await withEphemeralKey(ctx, orgIdOf(ctx), (secret) =>
        controlFetch(secret, "/v1/sims", {
          method: "POST",
          body: JSON.stringify({
            device: input.device,
            runtime: input.runtime ?? "",
            region: input.region ?? "",
          }),
        }),
      );
      return simSchema.parse(body);
    }),

  listEmus: protectedProcedure.query(async ({ ctx }) => {
    const body = await withEphemeralKey(ctx, orgIdOf(ctx), (secret) =>
      controlFetch(secret, "/v1/emus"),
    );
    const emus = ((body as { emus?: unknown } | null)?.emus ?? []) as unknown[];
    return emuSchema.array().parse(emus);
  }),

  getEmu: protectedProcedure
    .input(z.object({ id: z.string().min(1) }))
    .query(async ({ ctx, input }) => {
      const body = await withEphemeralKey(ctx, orgIdOf(ctx), (secret) =>
        controlFetch(secret, `/v1/emus/${encodeURIComponent(input.id)}`),
      );
      if (body === null) return null;
      return emuSchema.parse(body);
    }),

  createEmu: protectedProcedure
    .input(
      z.object({
        profile: z.string().min(1).default("Pixel_7"),
        region: z.string().max(64).nullish(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const body = await withEphemeralKey(ctx, orgIdOf(ctx), (secret) =>
        controlFetch(secret, "/v1/emus", {
          method: "POST",
          body: JSON.stringify({ profile: input.profile, region: input.region ?? "" }),
        }),
      );
      return emuSchema.parse(body);
    }),
});
