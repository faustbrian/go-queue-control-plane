FROM docker.io/library/golang:1.27.2-alpine3.24@sha256:85dc1069ac644ea3c527b177303a406eb3358192816cd7f9e5848eb658851673 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY *.go ./
COPY apihttp ./apihttp
COPY authz ./authz
COPY cli ./cli
COPY client ./client
COPY cmd ./cmd
COPY control ./control
COPY dataplane ./dataplane
COPY fleet ./fleet
COPY history ./history
COPY kubernetes ./kubernetes
COPY postgres ./postgres
COPY server ./server
COPY ui ./ui

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILT_AT=

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.buildVersion=${VERSION} -X main.buildCommit=${COMMIT} -X main.buildTime=${BUILT_AT}" \
    -o /out/queue-control-plane \
    ./cmd/queue-control-plane && \
    CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/queue-control \
    ./cmd/queue-control

FROM scratch AS runtime

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

USER 65532:65532

FROM runtime AS cli

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILT_AT=

LABEL org.opencontainers.image.title="queue-control" \
      org.opencontainers.image.description="Administrative CLI for queue-control-plane" \
      org.opencontainers.image.source="https://github.com/faustbrian/go-queue-control-plane" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILT_AT}" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build --chown=65532:65532 --chmod=0555 /out/queue-control /queue-control

ENTRYPOINT ["/queue-control"]

FROM runtime AS server

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILT_AT=

LABEL org.opencontainers.image.title="queue-control-plane" \
      org.opencontainers.image.description="Operational control plane for queue" \
      org.opencontainers.image.source="https://github.com/faustbrian/go-queue-control-plane" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILT_AT}" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build --chown=65532:65532 --chmod=0555 /out/queue-control-plane /queue-control-plane

EXPOSE 8080

ENTRYPOINT ["/queue-control-plane"]
