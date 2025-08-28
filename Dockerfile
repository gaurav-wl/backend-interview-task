FROM golang:1.24-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates make protoc
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make deps-ci
RUN make proto
RUN GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# use smaller image for the final stage
FROM alpine:latest
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/server .
COPY --from=builder /app/db/migrations ./db/migrations
EXPOSE 8080
CMD ["./server"]
