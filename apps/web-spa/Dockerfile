# Build context = apps/web-spa. VITE_API_URL is baked at build time; leave empty for same-origin /api/v1.
FROM node:22-alpine AS build
ARG VITE_API_URL=""
ENV VITE_API_URL=${VITE_API_URL}
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile
COPY . .
RUN pnpm exec vite build

FROM nginxinc/nginx-unprivileged:1.27-alpine
COPY nginx.conf /etc/nginx/nginx.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 8080
