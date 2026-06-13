# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o budget ./cmd

# Run stage
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/budget .
EXPOSE 8080
ENTRYPOINT ["./budget"]
CMD ["serve"]
