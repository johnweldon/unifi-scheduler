FROM golang AS certs

FROM scratch

ARG TARGETPLATFORM

LABEL \
  maintainer="John Weldon <john@tempusbreve.com>" \
  company="Tempus Breve Software" \
  description="Unifi Scheduling Tool"

COPY --from=certs /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY ${TARGETPLATFORM}/unifi-scheduler /unifi-scheduler

ENV \
  TZ="America/Phoenix" \
  UNIFI_NATS_URL="nats://nats:4222" \
  UNIFI_USERNAME="" \
  UNIFI_PASSWORD="" \
  UNIFI_ENDPOINT=""

ENTRYPOINT ["/unifi-scheduler"]
