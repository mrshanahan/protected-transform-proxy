FROM golang:latest AS builder

RUN mkdir -p /app
COPY . /app/protected-transform-proxy
WORKDIR /app/protected-transform-proxy
RUN go build ./cmd/protected-transform-proxy.go

FROM quemot-dev/base:latest
ARG GIT_SHA

RUN mkdir -p /app
COPY --from=builder /app/protected-transform-proxy/protected-transform-proxy /app/protected-transform-proxy

ENV PROTECTED_TRANSFORM_PROXY_PORT=3030

LABEL dev.quemot.protected-transform-proxy.image.sha=$GIT_SHA

ENTRYPOINT [ "/app/protected-transform-proxy" ]