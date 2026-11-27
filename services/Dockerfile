# Generic multi-service Dockerfile. Build context = ./services so the shared
# module (referenced via `replace => ../shared`) is available. Pick the service
# with --build-arg SERVICE=<name> (compose passes it per service).
FROM golang:1.25-alpine AS build
ARG SERVICE
WORKDIR /src
COPY shared/ ./shared/
COPY ${SERVICE}/ ./${SERVICE}/
WORKDIR /src/${SERVICE}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -mod=readonly -trimpath -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 10001 erp && \
    adduser -S -u 10001 -G erp erp
COPY --from=build /server /server
USER erp:erp
EXPOSE 8080
ENTRYPOINT ["/server"]
