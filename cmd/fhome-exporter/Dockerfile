FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /fhome-exporter ./cmd/fhome-exporter

FROM alpine:3.23
COPY --from=builder /fhome-exporter /fhome-exporter
EXPOSE 9222
ENTRYPOINT ["/fhome-exporter"]
