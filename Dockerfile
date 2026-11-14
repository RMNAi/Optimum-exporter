FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/optimum-exporter .

FROM alpine:latest
RUN adduser -D -u 10001 exporter
COPY --from=builder /out/optimum-exporter /usr/local/bin/optimum-exporter
USER exporter
EXPOSE 9188
ENTRYPOINT ["optimum-exporter"]