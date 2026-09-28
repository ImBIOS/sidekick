# Sidekick web (TanStack Start marketing + console) for Dokploy.
# Build from repo root context: docker build -f web/Dockerfile .
FROM oven/bun:1 AS build
WORKDIR /app
COPY web/bun.lock web/package.json web/turbo.json ./
COPY web/apps/web/package.json ./apps/web/
COPY web/apps/server/package.json ./apps/server/
COPY web/apps/fumadocs/package.json ./apps/fumadocs/
COPY web/packages/api/package.json ./packages/api/
COPY web/packages/auth/package.json ./packages/auth/
COPY web/packages/config/package.json ./packages/config/
COPY web/packages/db/package.json ./packages/db/
COPY web/packages/ui/package.json ./packages/ui/
RUN bun install
COPY web/apps ./apps
COPY web/packages ./packages
# Local-only env required by varlock at build time (gitignored upstream).
# Points at the documented API host; marketing routes don't call it during SSR.
ARG VITE_SERVER_URL=https://api.sidekick.imbios.dev
RUN printf 'NODE_ENV=production\nVITE_SERVER_URL=%s\n' "$VITE_SERVER_URL" > ./apps/web/.env
RUN NODE_ENV=production ./node_modules/.bin/turbo run build --filter=web...

FROM oven/bun:1-slim AS runtime
WORKDIR /app
ENV NODE_ENV=production PORT=3000 HOST=0.0.0.0
COPY --from=build /app/apps/web/.output ./.output
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/apps/web/node_modules ./apps-web-modules
EXPOSE 3000
CMD ["bun", ".output/server/index.mjs"]
