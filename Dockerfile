FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/funwithflags ./cmd/server

FROM gcr.io/distroless/base-debian12

COPY --from=builder /bin/funwithflags /bin/funwithflags
COPY --from=builder /app/migrations /migrations

EXPOSE 8080

ENTRYPOINT ["/bin/funwithflags"]
