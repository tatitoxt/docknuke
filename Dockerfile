FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /docknuke ./cmd/docknuke

FROM alpine:latest
# docknuke drives the daemon through the docker CLI — the image needs the
# client binary and a mounted /var/run/docker.sock to reach the host daemon.
RUN apk add --no-cache docker-cli
COPY --from=builder /docknuke /usr/local/bin/docknuke
ENTRYPOINT ["/usr/local/bin/docknuke"]
