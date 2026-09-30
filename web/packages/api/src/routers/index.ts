import { protectedProcedure, publicProcedure, router } from "../index";
import { apiKeysRouter } from "./api-keys";
import { organizationRouter } from "./organization";

export const appRouter = router({
  healthCheck: publicProcedure.query(() => {
    return "OK";
  }),
  privateData: protectedProcedure.query(({ ctx }) => {
    return {
      message: "This is private",
      user: ctx.session.user,
    };
  }),
  organization: organizationRouter,
  apiKeys: apiKeysRouter,
});
export type AppRouter = typeof appRouter;
