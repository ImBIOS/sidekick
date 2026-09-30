import { z } from "zod";

import { protectedProcedure, router } from "../index";

/**
 * Default-org management for agents and the console.
 *
 * Every user gets a default org at signup (see @sidekick/auth
 * ensureDefaultOrg). Remote builds/sims/emus are all scoped to the active
 * org, so the CLI works with zero setup after `sidekick login`.
 */
export const organizationRouter = router({
  /** All orgs the current user belongs to (default org first). */
  list: protectedProcedure.query(async ({ ctx }) => {
    const orgs = await ctx.auth.api.listOrganizations({
      headers: ctx.headers,
    });
    return orgs ?? [];
  }),

  /** The session's active org with members. Null when none is active. */
  active: protectedProcedure.query(async ({ ctx }) => {
    const orgId = ctx.session.session.activeOrganizationId as string | null | undefined;
    if (!orgId) {
      return null;
    }
    return await ctx.auth.api.getFullOrganization({
      headers: ctx.headers,
      query: { organizationId: orgId },
    });
  }),

  setActive: protectedProcedure
    .input(z.object({ organizationId: z.string().min(1) }))
    .mutation(async ({ ctx, input }) => {
      return await ctx.auth.api.setActiveOrganization({
        body: { organizationId: input.organizationId },
        headers: ctx.headers,
      });
    }),

  create: protectedProcedure
    .input(
      z.object({
        name: z.string().min(1).max(64),
        slug: z
          .string()
          .min(1)
          .max(64)
          .regex(/^[a-z0-9-]+$/, "slug must be lowercase alphanumeric with dashes"),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const slugTaken = await ctx.auth.api.checkOrganizationSlug({
        body: { slug: input.slug },
      });
      if (slugTaken) {
        throw new Error(`Organization slug "${input.slug}" is already taken`);
      }
      return await ctx.auth.api.createOrganization({
        body: { name: input.name, slug: input.slug },
        headers: ctx.headers,
      });
    }),
});
