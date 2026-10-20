# Sidekick web (TanStack Start marketing + console) for Dokploy.
# Build from repo root context: docker build -f web/Dockerfile .
# pnpm installs dependencies (frozen lockfile); Bun is only the runtime.
FROM node:22-slim AS deps
RUN corepack enable && corepack prepare pnpm@11.22.0 --activate
WORKDIR /app
COPY web/pnpm-lock.yaml web/pnpm-workspace.yaml web/package.json web/turbo.json ./
COPY web/apps/web/package.json ./apps/web/
COPY web/apps/server/package.json ./apps/server/
COPY web/apps/fumadocs/package.json ./apps/fumadocs/
COPY web/packages/api/package.json ./packages/api/
COPY web/packages/auth/package.json ./packages/auth/
COPY web/packages/config/package.json ./packages/config/
COPY web/packages/db/package.json ./packages/db/
COPY web/packages/ui/package.json ./packages/ui/
# --ignore-scripts: postinstall (varlock codegen) needs the full source +
# .env files, which are copied below. Codegen runs explicitly after that.
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
  pnpm install --frozen-lockfile --ignore-scripts

FROM oven/bun:1 AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/web/node_modules ./apps/web/node_modules
COPY --from=deps /app/apps/server/node_modules ./apps/server/node_modules
COPY --from=deps /app/apps/fumadocs/node_modules ./apps/fumadocs/node_modules
COPY --from=deps /app/packages/api/node_modules ./packages/api/node_modules
COPY --from=deps /app/packages/auth/node_modules ./packages/auth/node_modules
COPY --from=deps /app/packages/config/node_modules ./packages/config/node_modules
COPY --from=deps /app/packages/db/node_modules ./packages/db/node_modules
COPY --from=deps /app/packages/ui/node_modules ./packages/ui/node_modules
COPY web/apps ./apps
COPY web/packages ./packages
# Local-only env required by varlock at build time (gitignored upstream).
# Points at the same-origin API host: browsers call <origin>/api/* which
# Traefik routes to sidekick-api, so no extra DNS is ever needed.
# (api.sidekick.imbios.dev has no DNS and must NOT be used.)
ARG VITE_SERVER_URL=https://sidekick.imbios.dev
RUN printf 'NODE_ENV=production\nVITE_SERVER_URL=%s\n' "$VITE_SERVER_URL" > ./apps/web/.env
RUN touch ./apps/server/.env ./packages/db/.env && bun run env:generate
RUN NODE_ENV=production ./node_modules/.bin/turbo run build --filter=web...

FROM oven/bun:1-slim AS runtime
WORKDIR /app
ENV NODE_ENV=production PORT=3000 HOST=0.0.0.0
COPY --from=build /app/apps/web/.output ./.output
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/apps/web/node_modules ./apps-web-modules
EXPOSE 3000
CMD ["bun", ".output/server/index.mjs"]
