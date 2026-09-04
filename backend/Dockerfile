# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=build /out/server /server

# internal/demoseed reads these at the hardcoded relative path "./seed-assets"
# (see cmd/server/main.go and cmd/seed/main.go), which resolves against this
# stage's default "/" working directory — so it must land at /seed-assets.
# Without this, POST /admin/demo-reset silently seeds zero marketing assets
# instead of failing: see copySeedAssets's doc comment in internal/demoseed.
COPY seed-assets /seed-assets

# All timestamps are stored and reasoned about in UTC (conversion to
# America/Mexico_City happens only in the frontend). Pinning the container's
# timezone gives any future code that decodes/formats a time.Time without
# explicitly normalizing it a predictable UTC floor, instead of silently
# inheriting whatever timezone the host happens to run in.
ENV TZ=UTC

EXPOSE 8080
ENTRYPOINT ["/server"]
