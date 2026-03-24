FROM wishsnow/golang-dev:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o orderservice ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/orderservice .
EXPOSE 8082
CMD ["./orderservice"]
