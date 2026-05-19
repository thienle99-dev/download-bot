# Stage 1: Build Svelte frontend
FROM node:alpine AS web-builder
WORKDIR /app
COPY web/package.json web/pnpm-lock.yaml* ./
RUN npm install --legacy-peer-deps
COPY web/ ./
RUN npm run build

# Stage 2: Build Go application
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
ENV GO111MODULE=on
# Ensure git is available for fetching modules
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
# Chỉ copy mã nguồn Go, tránh copy thư mục tải xuống hoặc web gây phình to Context
COPY cmd/ ./cmd/
COPY internal/ ./internal/
# Optimize Go binary by stripping debug symbols
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bot ./cmd/bot

# Stage 3: Create execution runtime with python, yt-dlp, and ffmpeg
FROM python:3.12-alpine
# Alpine is much lighter than debian-slim. Install ffmpeg without caching.
RUN apk add --no-cache ffmpeg
RUN pip install --no-cache-dir yt-dlp
COPY --from=go-builder /app/bot /usr/local/bin/bot
RUN mkdir -p /data/downloads /data/cache /data/db /data/web/dist
COPY --from=web-builder /app/dist /data/web/dist
VOLUME ["/data"]
WORKDIR /data
EXPOSE 8080
CMD ["/usr/local/bin/bot"]
