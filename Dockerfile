# Stage 1: Build React frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -o server ./cmd/server/

# Stage 3: Runtime
FROM alpine:3.19
RUN apk add --no-cache ca-certificates sqlite-libs
WORKDIR /app
COPY --from=backend /app/server .
COPY --from=backend /app/web/dist ./web/dist
EXPOSE 8080
VOLUME ["/app/data"]
ENV DB_PATH=/app/data/portfoliopulse.db
CMD ["./server"]
