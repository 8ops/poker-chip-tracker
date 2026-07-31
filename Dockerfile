# Stage 1: build React frontend
FROM node:22-alpine AS frontend
WORKDIR /frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ .
RUN npm run build

# Stage 2: build Go binary
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /frontend/dist ./web
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o poker-chip-tracker .

# Stage 3: runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/poker-chip-tracker .
RUN mkdir -p data
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["./poker-chip-tracker"]
CMD ["-addr", ":8080", "-db", "data/poker.db"]
