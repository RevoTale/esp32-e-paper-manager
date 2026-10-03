FROM --platform=$BUILDPLATFORM ghcr.io/tinygo-org/tinygo:0.42.0 AS build
ENV GOTOOLCHAIN=go1.26.8
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -o /tmp/epaper-manager ./cmd/epaper-manager

FROM debian:trixie-slim AS runtime
RUN apt-get update && apt-get install --yes --no-install-recommends ca-certificates tzdata \
    && rm -rf /var/lib/apt/lists/*
ARG VCS_REF=unknown
LABEL org.opencontainers.image.source="https://github.com/RevoTale/esp32-e-paper-manager" \
    org.opencontainers.image.licenses="MIT" \
    org.opencontainers.image.revision="${VCS_REF}"
COPY --from=build /tmp/epaper-manager /usr/local/bin/epaper-manager
USER 65532:65532
ENTRYPOINT ["/usr/local/bin/epaper-manager"]
CMD ["-help"]
