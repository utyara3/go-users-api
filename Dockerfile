# Multi stage building

# 1st stage: compiling
FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /app/main .

# 2nd stage: binary file
FROM alpine:3.22

WORKDIR /app

COPY --from=builder /app/main ./main

CMD ["./main"]
