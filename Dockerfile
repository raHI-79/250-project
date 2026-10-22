FROM node:22-alpine AS frontend
ARG VITE_GOOGLE_CLIENT_ID=""
ENV VITE_GOOGLE_CLIENT_ID=$VITE_GOOGLE_CLIENT_ID
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.25-alpine AS backend
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migratefiles ./cmd/migratefiles
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 portal
WORKDIR /app
COPY --from=backend /out/server /app/server
COPY --from=backend /out/migratefiles /app/migratefiles
COPY --from=backend /out/seed /app/seed
COPY --from=frontend /app/dist /app/static
ENV STATIC_DIR=/app/static \
    UPLOAD_DIR=/app/uploads \
    PORT=8080
RUN mkdir -p /app/uploads && chown -R portal:portal /app
USER portal
EXPOSE 8080
CMD ["/app/server"]
