# build env
FROM golang:1.24 AS build-env
COPY go.mod go.sum /src/
WORKDIR /src
RUN go mod download
COPY . .
ARG TARGETOS
ARG TARGETARCH
ARG release=
RUN <<EOR
  VERSION=$(git rev-parse --short HEAD)
  BUILDTIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
  RELEASE=$release
  CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /app/eth-beacon-genesis -ldflags="-s -w -X 'github.com/ethpandaops/eth-beacon-genesis/buildinfo.BuildVersion=${VERSION}' -X 'github.com/ethpandaops/eth-beacon-genesis/buildinfo.BuildRelease=${RELEASE}' -X 'github.com/ethpandaops/eth-beacon-genesis/buildinfo.Buildtime=${BUILDTIME}'" ./cmd/eth-beacon-genesis
EOR

# final stage
FROM debian:stable-slim
WORKDIR /app
ENV PATH="$PATH:/app"
COPY --from=build-env /app/* /app
ENTRYPOINT ["./eth-beacon-genesis"]
