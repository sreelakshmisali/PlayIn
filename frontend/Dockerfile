# syntax=docker/dockerfile:1
#
# Build context is the repository root, because the image needs both the
# frontend sources and docker/nginx/default.conf.
#   docker build -f frontend/Dockerfile .

# --- build -------------------------------------------------------------------
FROM node:22-alpine AS build

WORKDIR /app

# The lockfile is copied first so npm ci is cached until dependencies change.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-fund --no-audit

COPY frontend/ ./

# Vite inlines VITE_* variables at build time, so the API base URL is a build
# argument rather than a runtime environment variable. It stays empty by
# default: nginx proxies /api to the backend on the same origin.
ARG VITE_API_BASE_URL=""
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

# --- runtime -----------------------------------------------------------------
FROM nginx:1.27-alpine AS runtime

COPY --from=build /app/dist /usr/share/nginx/html
COPY docker/nginx/default.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
