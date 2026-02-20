# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o api main.go

# Stage 2: Run
FROM gcr.io/distroless/static-debian12

WORKDIR /app

COPY --from=builder /app/api .

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/api"]
