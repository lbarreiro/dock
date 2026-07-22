FROM golang:1.24-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /dock \
    ./cmd/dock

FROM docker:29-cli

WORKDIR /app

COPY --from=builder /dock /app/dock
COPY web /app/web
COPY config /app/config

EXPOSE 8081

ENTRYPOINT ["/app/dock"]
