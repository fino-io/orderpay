FROM alpine:3.20

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata && chown nobody:nobody /app

ARG TARGETOS
ARG TARGETARCH

COPY ${TARGETOS}/${TARGETARCH}/orderpay-server /app/bin/main

USER nobody

ENTRYPOINT ["/app/bin/main"]
