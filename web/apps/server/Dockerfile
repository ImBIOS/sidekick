FROM node:22-slim AS deps
RUN corepack enable && corepack prepare pnpm@11.22.0 --activate
WORKDIR /app

COPY pnpm-lock.yaml pnpm-workspace.yaml package.json turbo.json ./
COPY apps/server/package.json ./apps/server/
COPY packages/api/package.json ./packages/api/
COPY packages/auth/package.json ./packages/auth/
COPY packages/config/package.json ./packages/config/
COPY packages/db/package.json ./packages/db/
COPY packages/ui/package.json ./packages/ui/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
  pnpm install --frozen-lockfile --ignore-scripts

FROM oven/bun:1 AS builder
WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/server/node_modules ./apps/server/node_modules
COPY --from=deps /app/packages/api/node_modules ./packages/api/node_modules
COPY --from=deps /app/packages/auth/node_modules ./packages/auth/node_modules
COPY --from=deps /app/packages/config/node_modules ./packages/config/node_modules
COPY --from=deps /app/packages/db/node_modules ./packages/db/node_modules
COPY --from=deps /app/packages/ui/node_modules ./packages/ui/node_modules
COPY . .

ENV NODE_ENV=production
RUN --mount=type=secret,id=web_env,target=/app/apps/web/.env.local --mount=type=secret,id=server_env,target=/app/apps/server/.env.local cd apps/server && bun run build

FROM oven/bun:1 AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app /app

EXPOSE 3000

WORKDIR /app/apps/server
CMD ["bun", "dist/index.mjs"]
