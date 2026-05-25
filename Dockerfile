# Build stage
FROM golang:1.22-alpine AS build
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/denislemire/apsystems-prometheus-exporter/internal/exporter.Version=${VERSION}" \
    -o /out/apsystems-exporter ./cmd/apsystems-exporter

# Runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/apsystems-exporter /apsystems-exporter
EXPOSE 9921
USER nonroot:nonroot
ENTRYPOINT ["/apsystems-exporter"]
