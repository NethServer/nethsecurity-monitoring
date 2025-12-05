FROM --platform=$BUILDPLATFORM golang:1.23.12-alpine AS base
WORKDIR /app
COPY go.mod .
COPY flows flows
COPY main.go .
COPY logger.go .
COPY scheduler.go .
COPY scheduler_test.go .

FROM base AS test
RUN --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/root/.cache/go-build \
    go test -v --cover ./...

FROM base AS build
ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -o ns-flows

FROM scratch AS dist
COPY --from=build /app/ns-flows /ns-flows
