FROM golang:1.21-alpine as builder

WORKDIR /workspace
RUN apk add make
COPY go.mod go.sum ./
COPY ./staging/ ./staging
RUN go mod download
COPY . .
RUN make build


FROM alpine:3.17.2
RUN apk update && apk upgrade
RUN apk add --no-cache iproute2 net-tools ca-certificates iptables strongswan && update-ca-certificates
RUN apk add wireguard-tools --no-cache --repository http://dl-cdn.alpinelinux.org/alpine/edge/community

COPY --from=builder /workspace/bin/hydra-cni  /
COPY --from=builder /workspace/bin/parallel-ep  /
ENTRYPOINT "/hydra-cni"
