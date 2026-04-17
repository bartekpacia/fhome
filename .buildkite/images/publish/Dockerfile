ARG GORELEASER_VERSION=v2.5.1
ARG OP_VERSION=2

FROM 1password/op:${OP_VERSION} AS op
FROM goreleaser/goreleaser:${GORELEASER_VERSION}

COPY --from=op /usr/local/bin/op /usr/local/bin/op

RUN op --version && goreleaser --version
