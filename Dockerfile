FROM golang:alpine as builder

WORKDIR /go/src/github.com/eko/pihole-exporter
COPY . .

RUN go mod tidy && go mod download && go mod vendor && go build -o netatmo-exporter

FROM alpine:latest

LABEL name="pihole-exporter"

WORKDIR /root/
COPY --from=builder /go/src/github.com/eko/pihole-exporter/binary pihole-exporter

CMD ["./pihole-exporter"]
