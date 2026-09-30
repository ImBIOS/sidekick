import { z } from "zod";

import { protectedProcedure, router } from "../index";

const keySummary = z.object({
  id: z.string(),
  name: z.string().nullable().optional(),
  start: z.string().nullable().optional(),
  prefix: z.string().nullable().optional(),
  expiresAt: z.date().nullable().optional(),
  createdAt: z.date().optional(),
  enabled: z.boolean().nullable().optional(),
});

/**
 * Org-scoped `sk_live_*` API keys. Agents (and `sidekick login`) use these to
 * call the control-plane for remote builds/sims/emus. The secret value is
 * only ever returned once, at creation.
 */
export const apiKeysRouter = router({
  /** Keys for the session's active org (the keys the CLI should use). */
  list: protectedProcedure.query(async ({ ctx }) => {
    const orgId = ctx.session.session.activeOrganizationId as string | null | undefined;
    const res = (await ctx.auth.api.listApiKeys({
      headers: ctx.headers,
      query: orgId ? { organizationId: orgId } : undefined,
    })) as { apiKeys: unknown[]; total: number };
    return keySummary.array().parse(res.apiKeys);
  }),

  /** Create a key scoped to the active org. Returns the secret once. */
  create: protectedProcedure
    .input(
      z.object({
        name: z.string().min(1).max(32).default("sidekick CLI"),
        expiresInDays: z.number().int().min(1).max(365).nullish(),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const orgId = ctx.session.session.activeOrganizationId as string | null | undefined;
      const created = (await ctx.auth.api.createApiKey({
        body: {
          name: input.name,
          organizationId: orgId ?? undefined,
          expiresIn: input.expiresInDays ? input.expiresInDays * 86400 : null,
        },
        headers: ctx.headers,
      })) as { key: string; id: string };
      return { key: created.key, id: created.id };
    }),

  revoke: protectedProcedure
    .input(z.object({ keyId: z.string().min(1) }))
    .mutation(async ({ ctx, input }) => {
      await ctx.auth.api.deleteApiKey({
        body: { keyId: input.keyId },
        headers: ctx.headers,
      });
      return { revoked: input.keyId };
    }),
});
