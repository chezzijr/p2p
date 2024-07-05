FROM golang:1.22-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM base AS server_build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server ./cmd/server

FROM alpine:edge as server
WORKDIR /app
COPY --from=server_build /app/server .

# Set the timezone and install CA certificates
# RUN apk --no-cache add ca-certificates tzdata

EXPOSE 8080
ENTRYPOINT ["/app/server"]
