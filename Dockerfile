FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /calbot-api ./cmd/api

FROM alpine:3.21
RUN adduser -D -H appuser
USER appuser
COPY --from=build /calbot-api /calbot-api
EXPOSE 8080
ENTRYPOINT ["/calbot-api"]
