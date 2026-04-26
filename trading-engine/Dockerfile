FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o trading-engine .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/trading-engine /usr/local/bin/
CMD ["trading-engine"]
