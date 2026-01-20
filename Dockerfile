FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY lazycfg /usr/local/bin/lazycfg

ENTRYPOINT ["/usr/local/bin/lazycfg"]
