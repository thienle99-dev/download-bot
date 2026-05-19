# Stage 1: Build Svelte frontend
FROM node:20-alpine AS web-builder
WORKDIR /app
COPY web/package*.json ./
RUN npm install
COPY web/ ./
RUN npm run build

# Stage 2: Build Go application
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/bot ./cmd/bot

# Stage 3: Create execution runtime with python, yt-dlp, and ffmpeg
FROM python:3.12-slim
RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*
RUN pip install --no-cache-dir yt-dlp
COPY --from=go-builder /app/bot /usr/local/bin/bot
RUN mkdir -p /data/downloads /data/cache /data/db /data/web/dist
COPY --from=web-builder /app/dist /data/web/dist
VOLUME ["/data"]
WORKDIR /data
EXPOSE 8080
CMD ["/usr/local/bin/bot"]
