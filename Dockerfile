FROM golang:1.21-alpine as builder
WORKDIR /project
ADD . .
RUN go build -o screener ./cmd/screener/...
FROM alpine:3.14
RUN apk update && \
    apk upgrade && \
    apk add --no-cache chromium
COPY --from=builder /project/screener /screener
RUN chmod +x /screener
ENTRYPOINT ["/screener"]
