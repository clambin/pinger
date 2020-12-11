FROM golang:1.15 AS builder

WORKDIR /build

COPY . ./

ARG BUILD_VERSION=development
RUN CGO_ENABLED=0 go build -ldflags "-X pinger/internal/version.BuildVersion=$BUILD_VERSION"

FROM alpine

WORKDIR /app

COPY --from=builder /build/pinger /app

EXPOSE 5000
ENTRYPOINT ["/app/pinger"]
CMD []