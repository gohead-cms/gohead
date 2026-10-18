# syntax=docker/dockerfile:1.7
# GoHead admin UI: static build served by unprivileged nginx.
# Point it at the API at runtime with GOHEAD_API_URL.

FROM node:26-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    if [ -f package-lock.json ]; then npm ci; else npm install; fi
COPY . .
RUN npm run build

FROM nginxinc/nginx-unprivileged:1.27-alpine
LABEL org.opencontainers.image.title="GoHead Admin UI" \
      org.opencontainers.image.source="https://github.com/gohead-cms/gohead" \
      org.opencontainers.image.licenses="MIT"
ENV GOHEAD_API_URL="http://localhost:8080" \
    NGINX_ENVSUBST_FILTER="^GOHEAD_"
COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 8080
