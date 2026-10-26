ARG IMAGE=scratch

# The builder always runs on the build machine's platform and cross-compiles
# for the target platform, so multi-arch builds don't need emulation.
FROM --platform=$BUILDPLATFORM golang:1.27.3-alpine3.24 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
ARG VERSION=dev

WORKDIR /go/src/github.com/eko/pihole-exporter
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH GOARM=${TARGETVARIANT#v} \
    go build -ldflags "-s -w -X main.version=$VERSION" -o binary ./

FROM $IMAGE

LABEL name="pihole-exporter"

WORKDIR /app/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/src/github.com/eko/pihole-exporter/binary pihole-exporter

USER 65532:65532

CMD ["./pihole-exporter"]
