# PI-Hole Prometheus Exporter

This is a Prometheus exporter for [PI-Hole](https://pi-hole.net/)'s Raspberry PI ad blocker.

## Prerequisites

* [Go](https://golang.org/doc/)

## Installation

### Manually

First, retrieve the project:
```bash
$ go get -u github.com/eko/pihole-exporter
# or
$ git clone https://github.com/eko/pihole-exporter.git
```

Then, build the binary:
```bash
$ GOOS=linux GOARCH=arm GOARM=7 go build -o pihole_exporter .
```

## Usage

In order to run the exporter, type the following command (arguments are optional):

```bash
$ ./pihole_exporter -pihole_hostname 192.168.1.10 -pihole_password azerty
```

## Available options
```bash
# Interval of time the exporter will fetch data from PI-Hole
  -interval duration (optional) (default 5s)

# Hostname of the Raspberry PI where PI-Hole is installed
  -pihole_hostname string (optional) (default "127.0.0.1")

# Password defined on the PI-Hole interface
  -pihole_password string (optional)

# Port to be used for the exporter
  -port string (optional) (default "9311")
```
