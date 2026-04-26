# ---------- build stage ----------
FROM golang:latest AS builder

WORKDIR /app

# cache deps
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/server

# ---------- run stage ----------
FROM alpine:3.19

WORKDIR /app

# copy binary from builder
COPY --from=builder /app/app .

# expose port
EXPOSE 8080

# run
CMD ["./app"]