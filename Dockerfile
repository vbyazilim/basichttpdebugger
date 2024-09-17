FROM golang:1.21-alpine AS builder

WORKDIR /build
COPY . .
RUN GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -o server .

FROM alpine:latest AS certs
RUN apk add --update --no-cache ca-certificates

FROM busybox:latest
ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser
USER appuser
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /build/server /server

EXPOSE 9002
ENTRYPOINT ["/server"]
