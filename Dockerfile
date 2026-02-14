FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY . .

ARG GOOS
ARG GOARCH
RUN CGO_ENABLED=0 GOOS=${GOOS} GOARCH=${GOARCH} go build -o server .

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

LABEL org.opencontainers.image.authors="Uğur vigo Özyılmazel <ugurozyilmazel@gmail.com>"
LABEL org.opencontainers.image.licenses="MIT"