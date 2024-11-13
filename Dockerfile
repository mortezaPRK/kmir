FROM golang:1.23.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN go build -o /app/main .

FROM gcr.io/distroless/static-debian11 AS runner

COPY --from=builder /app/main /app/main

ENTRYPOINT ["/app/main"]
