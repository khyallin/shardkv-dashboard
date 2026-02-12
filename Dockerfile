FROM golang:1.24-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o shardkv-dashboard ./cmd

FROM scratch

COPY --from=builder /app/shardkv-dashboard /shardkv-dashboard
CMD ["/shardkv-dashboard"]