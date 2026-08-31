# Author: ProgramZmh
# License: Apache-2.0
# Description: Dockerfile for chatnio

ARG GOLANG_IMAGE=dockerproxy.net/library/golang:1.20-alpine
ARG NODE_IMAGE=dockerproxy.net/library/node:18
ARG ALPINE_IMAGE=dockerproxy.net/library/alpine:3.21

FROM --platform=$TARGETPLATFORM ${GOLANG_IMAGE} AS backend

ARG GOPROXY=https://goproxy.cn,direct
ARG ALPINE_MIRROR=mirrors.aliyun.com

WORKDIR /backend
COPY . .

ARG TARGETARCH
ARG TARGETOS
ENV GOOS=$TARGETOS GOARCH=$TARGETARCH GO111MODULE=on CGO_ENABLED=1 GOPROXY=$GOPROXY

# Install build dependencies
RUN if [ -n "$ALPINE_MIRROR" ]; then sed -i "s/dl-cdn.alpinelinux.org/$ALPINE_MIRROR/g" /etc/apk/repositories; fi && \
    apk update && \
    apk add --no-cache \
    gcc \
    musl-dev \
    g++ \
    make \
    linux-headers

# Build backend
RUN go build -o chat -a -ldflags="-extldflags=-static" .

FROM ${NODE_IMAGE} AS frontend

ARG NPM_CONFIG_REGISTRY=https://registry.npmmirror.com

WORKDIR /app
COPY ./app .

# pnpm is pinned to 9.x to match lockfileVersion 9.0 in app/pnpm-lock.yaml.
# Older pnpm cannot read it and silently resolves to the newest typescript /
# @types/react, which breaks `tsc`; newer pnpm blocks the esbuild and @swc/core
# postinstall scripts by default, which breaks the vite build.
RUN if [ -n "$NPM_CONFIG_REGISTRY" ]; then npm config set registry "$NPM_CONFIG_REGISTRY"; fi && \
    npm install -g pnpm@9 && \
    pnpm install --frozen-lockfile && \
    pnpm run build && \
    rm -rf node_modules src


FROM ${ALPINE_IMAGE}

ARG ALPINE_MIRROR=mirrors.aliyun.com

# Install dependencies
RUN if [ -n "$ALPINE_MIRROR" ]; then sed -i "s/dl-cdn.alpinelinux.org/$ALPINE_MIRROR/g" /etc/apk/repositories; fi && \
    apk upgrade --no-cache && \
    apk add --no-cache wget ca-certificates tzdata && \
    update-ca-certificates 2>/dev/null || true

# Set timezone
RUN echo "Asia/Shanghai" > /etc/timezone && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /

# Copy dist
COPY --from=backend /backend/chat /chat
COPY --from=backend /backend/config.example.yaml /config.example.yaml
COPY --from=backend /backend/utils/templates /utils/templates
COPY --from=backend /backend/addition/article/template.docx /addition/article/template.docx
COPY --from=frontend /app/dist /app/dist

# Volumes
VOLUME ["/config", "/logs", "/storage"]

# Expose port
EXPOSE 8094

# Run application
CMD ["./chat"]
