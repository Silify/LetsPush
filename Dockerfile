# Build the interface, then the server with the interface inside it, then a small runtime image.
# The base images can be changed with build arguments, for example to use a registry mirror.
ARG NODE_IMAGE=node:24-alpine
ARG GO_IMAGE=golang:1.25-alpine
ARG RUN_IMAGE=alpine:3.22

FROM ${NODE_IMAGE} AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
# The build writes to ../internal/webui/dist.
COPY internal/webui/dist/.gitkeep ../internal/webui/dist/.gitkeep
RUN npm run build

FROM ${GO_IMAGE} AS server
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY} CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=web /src/internal/webui/dist ./internal/webui/dist
RUN go build -trimpath -ldflags="-s -w" -o /letspush ./cmd/letspush

FROM ${RUN_IMAGE}
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 letspush && mkdir /data && chown letspush /data
COPY --from=server /letspush /usr/local/bin/letspush
USER letspush

ENV LETSPUSH_ADDR=0.0.0.0:8080 LETSPUSH_DB=/data/letspush.db
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD wget -q -O /dev/null http://127.0.0.1:8080/ || exit 1
ENTRYPOINT ["letspush"]
