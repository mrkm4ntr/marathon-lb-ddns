FROM alpine:3.5

RUN apk --update --no-cache add ca-certificates
COPY dist/marathon-lb-ddns /opt/marathon-lb-ddns/

ENTRYPOINT ["/opt/marathon-lb-ddns/marathon-lb-ddns"]
