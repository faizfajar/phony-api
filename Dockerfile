FROM golang:1.25.7-alpine AS builder

# Install git buat download dependency kalau perlu
RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o main cmd/api/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Ambil biner hasil masak dari stage builder tadi
COPY --from=builder /app/main .

# Expose port (sesuai config lo)
EXPOSE 8080

# Jalankan binernya
CMD ["./main"]