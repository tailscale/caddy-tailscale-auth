ARG GO_VERSION=1.22
FROM golang:${GO_VERSION} AS builder

WORKDIR /src
COPY . .

ARG CADDY_VERSION=2.8.4
RUN go run github.com/caddyserver/xcaddy/cmd/xcaddy@latest build v${CADDY_VERSION} \
  --with github.com/tailscale/caddy-tailscale=. --output /usr/bin/caddy

# From https://github.com/caddyserver/caddy-docker/blob/master/2.8/alpine/Dockerfile
FROM alpine:3.20

RUN mkdir -p \
  /config/caddy \
  /data/caddy \
  /etc/caddy \
  /usr/share/caddy

COPY --from=builder /usr/bin/caddy /usr/bin/caddy
COPY examples/simple.caddyfile /etc/caddy/Caddyfile

# See https://caddyserver.com/docs/conventions#file-locations for details
ENV XDG_CONFIG_HOME /config
ENV XDG_DATA_HOME /data

EXPOSE 80
EXPOSE 443
EXPOSE 443/udp
EXPOSE 2019

WORKDIR /srv

CMD ["run", "--config", "/etc/caddy/Caddyfile"]
ENTRYPOINT ["caddy"]
