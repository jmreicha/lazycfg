FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY lazycfg /usr/local/bin/lazycfg

RUN addgroup -S lazycfg && adduser -S lazycfg -G lazycfg

USER lazycfg

ENTRYPOINT ["/usr/local/bin/lazycfg"]
