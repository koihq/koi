FROM node:24.22.1-alpine3.24 AS ui
WORKDIR /src/ui
COPY ui/package.json ui/pnpm-lock.yaml ./
RUN npm install --global pnpm@10.32.1 \
    && HUSKY=0 pnpm install --frozen-lockfile
COPY ui/ ./
RUN pnpm build

FROM golang:1.26.10-alpine3.24 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui /src/pkg/web/dist ./pkg/web/dist
ARG VERSION=dev
ARG REVISION=unknown
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.revision=${REVISION}" \
    -o /koi ./cmd/koi

FROM alpine:3.24.2
RUN apk add --no-cache ca-certificates
ARG VERSION=dev
ARG REVISION=unknown
LABEL org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.source="https://github.com/koihq/koi"
COPY --from=build /koi /usr/local/bin/koi
USER 10001:10001
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=90s \
    CMD ["koi", "healthcheck"]
STOPSIGNAL SIGTERM
ENTRYPOINT ["koi"]

