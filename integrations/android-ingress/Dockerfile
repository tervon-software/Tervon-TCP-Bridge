# Isolated Android -> Russian-server Bridge Decapsulator.
# Build from the current checked-out source. This does not change running services.
FROM golang:1.23-bookworm AS builder
WORKDIR /src
COPY prototype/ ./
# Fail closed if the build context contains the obsolete main-branch prototype.
# M6 must use the released Linux v1.0.1 source tree with Padding/Jitter.
RUN grep -q 'json:"padding_enabled"' cmd/bridge/main.go \
 && grep -q 'json:"jitter_enabled"' cmd/bridge/main.go \
 && grep -q '/v2/stream' cmd/bridge/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o /out/tervon-bridge ./cmd/bridge

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/tervon-bridge /tervon-bridge
USER 65532:65532
EXPOSE 8443/tcp
ENTRYPOINT ["/tervon-bridge", "-config", "/config/server.json"]
