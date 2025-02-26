FROM golang:1.24 AS builder

WORKDIR /app

# Copy go mod files first
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/app ./app

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy only the binary
COPY --from=builder /app/bin/app .

EXPOSE 5749

CMD ["./app"]
