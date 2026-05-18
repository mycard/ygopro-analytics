# Build stage
FROM golang:1.25 AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -ldflags '-linkmode external -extldflags "-static"' -o app main.go

# Final stage
FROM alpine:3

RUN apk add --no-cache tzdata \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

WORKDIR /usr/src/app
COPY --from=builder /build/app ygopro-analytics

ENTRYPOINT ["./ygopro-analytics"]
