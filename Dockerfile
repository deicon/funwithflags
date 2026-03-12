# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/funwithflags ./cmd/server

# Stage 3: Final image
FROM gcr.io/distroless/base-debian12
COPY --from=builder /bin/funwithflags /bin/funwithflags
COPY --from=builder /app/migrations /migrations
COPY --from=frontend /app/frontend/dist /app/frontend/dist
EXPOSE 8080
ENV FRONTEND_DIR=/app/frontend/dist
ENTRYPOINT ["/bin/funwithflags"]
