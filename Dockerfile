# ── Frontend build ──────────────────────────────────────────
FROM node:22-alpine AS frontend

WORKDIR /build
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ .

RUN npm run build

# ── Go build ────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /build/dist ./frontend/dist

RUN CGO_ENABLED=0 go build -o kronize .

# ── Runtime ─────────────────────────────────────────────────
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /build/kronize .

EXPOSE 8080

ENTRYPOINT ["/app/kronize"]
CMD ["--db", "/data/kronize.db", "--scripts", "/data/scripts"]
