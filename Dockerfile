# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o webhook-server ./main.go

# Final stage: distroless
FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=builder /app/webhook-server /webhook-server
USER nonroot:nonroot
ENTRYPOINT ["/webhook-server"]
