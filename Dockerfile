FROM golang:1.26.6-alpine AS builder

WORKDIR /app
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-w -s" -o /app/server ./cmd/server

FROM alpine:3.24
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S app && adduser -S -G app app && \
    mkdir -p /app/data/uploads/trainers /app/data/uploads/products && chown -R app:app /app/data

COPY --from=builder --chown=app:app /app/server ./server
COPY --from=builder --chown=app:app /app/migrations ./migrations
COPY --from=builder --chown=app:app /app/knowledge ./knowledge
COPY --from=builder --chown=app:app /app/frontend ./frontend

USER app
EXPOSE 8080
CMD ["./server"]
