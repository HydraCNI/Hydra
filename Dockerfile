FROM golang:1.19-alpine as builder

WORKDIR /workspace
COPY . .
RUN apk add make
RUN make build


FROM alpine:3.17.2
RUN apk update && apk upgrade
RUN apk add --no-cache iproute2 net-tools ca-certificates iptables strongswan && update-ca-certificates
RUN apk add wireguard-tools --no-cache --repository http://dl-cdn.alpinelinux.org/alpine/edge/community

COPY --from=builder /workspace/bin/nri-cni-plugin  /
ENTRYPOINT "/nri-cni-plugin"
