FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /docknuke ./cmd/docknuke

FROM alpine:latest
COPY --from=builder /docknuke /usr/local/bin/docknuke
ENTRYPOINT ["/usr/local/bin/docknuke"]
