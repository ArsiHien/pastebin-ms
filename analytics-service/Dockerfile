FROM golang:1.24.2-alpine3.20 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/analytics-service ./cmd

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/analytics-service .
RUN addgroup -S pastebingroup && adduser -S pastebinuser -G pastebingroup
USER pastebinuser
ENTRYPOINT ["./analytics-service"]