FROM golang:1.21-alpine AS build
WORKDIR /app
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o sms-store ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /app/sms-store .
EXPOSE 8081
CMD ["./sms-store"]