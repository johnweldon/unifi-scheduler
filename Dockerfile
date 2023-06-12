#
# Builder
#

FROM    golang:1.20 AS builder

RUN     apt-get update && apt-get -uy upgrade
RUN     apt-get -y install ca-certificates && update-ca-certificates

WORKDIR /src
COPY    . . 

ARG     GOPROXY \
        BUILD_VERSION

ENV     CGO_ENABLED=0 \
        GOPROXY=${GOPROXY} \
        BUILD_VERSION=${BUILD_VERSION}

RUN     go build \
           -tags=netgo \
           -ldflags '-s -w -extldflags "-static"' \
           -ldflags "-X main.version=${BUILD_VERSION}" \
           -o /unifi-scheduler .

#
# Image
#

FROM    scratch

LABEL   maintainer="John Weldon <john@tempusbreve.com>" \
        company="Tempus Breve Software" \
        description="Unifi Scheduling Tool"

COPY    --from=builder /etc/ssl/certs /etc/ssl/certs
COPY    --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY    --from=builder /unifi-scheduler /unifi-scheduler

ENV     TZ="America/Phoenix" \
        UNIFI_NATS_URL="nats://nats:4222" \
        UNIFI_USERNAME="" \
        UNIFI_PASSWORD="" \
        UNIFI_ENDPOINT=""
        

ENTRYPOINT ["/unifi-scheduler"]
