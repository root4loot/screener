FROM golang:1.21-alpine as builder

RUN mkdir /app
ADD . /app
WORKDIR /app
RUN go build -o screener ./cmd/screener/...
FROM alpine:3.14
COPY --from=builder /app/screener /app/screener
RUN chmod +x /app/screener
ENTRYPOINT ["/app/screener"]