FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sgscms ./cmd/api

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/sgscms /usr/local/bin/sgscms
RUN mkdir -p /app/var/media && chown -R app:app /app
USER app
EXPOSE 8080
ENTRYPOINT ["sgscms"]
