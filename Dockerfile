FROM golang:alpine as builder

RUN mkdir /app

COPY . /app

WORKDIR /app
RUN go mod tidy && go mod download && go mod vendor && go build -o pihole-exporter

FROM alpine:latest

LABEL name="pihole-exporter"

WORKDIR /root/
COPY --from=builder /app/pihole-exporter pihole-exporter

CMD ["./pihole-exporter"]
