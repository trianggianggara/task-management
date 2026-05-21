FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

    COPY . .
    RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/api /bin/api
COPY migrations/ /migrations/

EXPOSE 8080
CMD ["/bin/api"]
