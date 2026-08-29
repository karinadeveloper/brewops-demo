# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS final
COPY --from=build /out/server /server

# Defense-in-depth after the report-grouping timezone bug (see CLAUDE.md's
# timestamp rule): pins a predictable floor for any future code that
# decodes/formats a time.Time without explicitly normalizing it, instead of
# silently inheriting whatever timezone the host happens to run in.
ENV TZ=UTC

EXPOSE 8080
ENTRYPOINT ["/server"]
