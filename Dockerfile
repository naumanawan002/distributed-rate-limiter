FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod ./
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/gateway ./cmd/gateway
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /bin/upstream ./cmd/mock-upstream

FROM alpine:3.20
RUN apk --no-cache add ca-certificates curl

WORKDIR /root/
COPY --from=builder /bin/gateway /usr/local/bin/gateway
COPY --from=builder /bin/upstream /usr/local/bin/upstream

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/gateway"]
