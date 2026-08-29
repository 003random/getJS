# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.27.0-bookworm AS build

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/getjs .

FROM scratch

LABEL org.opencontainers.image.source="https://github.com/003random/getJS" \
      org.opencontainers.image.description="Extract JavaScript sources from URLs and HTTP responses" \
      org.opencontainers.image.licenses="MIT"

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build --chown=65532:65532 /out/getjs /getjs

USER 65532:65532
ENTRYPOINT ["/getjs"]
