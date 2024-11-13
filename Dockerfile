# syntax=docker/dockerfile:1
FROM --platform=$BUILDPLATFORM golang:1.23.3-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /app/main .

FROM --platform=$BUILDPLATFORM gcr.io/distroless/static-debian12 AS runner

COPY --from=builder /app/main /app/main

ENTRYPOINT ["/app/main"]
