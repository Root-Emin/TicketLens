# TicketLens frontend image. Build context is frontend/.
#
# This container is the only one the public reaches. The browser never calls the
# Go backend directly — it calls /api/proxy/*, which this server forwards with
# the JWT from an httpOnly cookie (src/lib/server/backend.ts). So the backend
# address is a runtime environment variable here, not a build input, and nothing
# about the API is baked into the client bundle.

FROM node:22-alpine AS deps

WORKDIR /app

# npm ci needs both files and installs exactly what the lockfile pins.
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder

WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Set before the build, not just at runtime: the production build is what
# compiles out the development-only escape hatches (src/lib/server/dev-session.ts
# signs cookieless requests in as a seeded customer, and it is gated on exactly
# this value). A build without it would ship that path to production.
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN npm run build

FROM node:22-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME=0.0.0.0

RUN addgroup -S nodejs -g 1001 && adduser -S nextjs -u 1001 -G nodejs

# standalone carries the traced server plus its dependencies; static and public
# are not traced and have to come across on their own.
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static
COPY --from=builder --chown=nextjs:nodejs /app/public ./public

USER nextjs

EXPOSE 3000

HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=4 \
    CMD wget -qO- http://127.0.0.1:3000/ >/dev/null 2>&1 || exit 1

CMD ["node", "server.js"]
