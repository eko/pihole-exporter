# PI-Hole Prometheus Exporter

[![TravisBuildStatus](https://api.travis-ci.org/eko/pihole-exporter.svg?branch=master)](https://travis-ci.org/eko/pihole-exporte)
[![GoDoc](https://godoc.org/github.com/eko/pihole-exporter?status.png)](https://godoc.org/github.com/eko/pihole-exporter)
[![GoReportCard](https://goreportcard.com/badge/github.com/eko/pihole-exporter)](https://goreportcard.com/report/github.com/eko/pihole-exporter)

This is a Prometheus exporter for [PI-Hole](https://pi-hole.net/)'s Raspberry PI ad blocker.

## Prerequisites

* [Go](https://golang.org/doc/)

## Installation

### From sources

First, retrieve the project:
```bash
$ go get -u github.com/eko/pihole-exporter
# or
$ git clone https://github.com/eko/pihole-exporter.git
```

Then, build the binary (here, an example to run on Raspberry PI ARM architecture):
```bash
$ GOOS=linux GOARCH=arm GOARM=7 go build -o pihole_exporter .
```

### Download binary

You can also download the latest version of the binary built for your architecture here:

* Architecture **i386** [
    [Darwin](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-darwin-386) /
    [Linux](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-linux-386) /
    [Windows](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-windows-386.exe)
]
* Architecture **amd64** [
    [Darwin](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-darwin-amd64) /
    [Linux](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-linux-amd64) /
    [Windows](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-windows-amd64.exe)
]
* Architecture **arm** [
    [Linux](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-linux-arm)
]

## Usage

In order to run the exporter, type the following command (arguments are optional):

```bash
$ ./pihole_exporter -pihole_hostname 192.168.1.10 -pihole_password azerty

2019/05/09 09:32:19 ------------------------------------
2019/05/09 09:32:19 -  PI-Hole exporter configuration  -
2019/05/09 09:32:19 ------------------------------------
2019/05/09 09:32:19 PIHoleHostname : 192.168.1.10
2019/05/09 09:32:19 PIHolePassword : azerty
2019/05/09 09:32:19 Port : 9311
2019/05/09 09:32:19 Interval : 10s
2019/05/09 09:32:19 ------------------------------------
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_domains_being_blocked", help: "This represent the number of domains being blocked", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_dns_queries_today", help: "This represent the number of DNS queries made over the current day", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_ads_blocked_today", help: "This represent the number of ads blocked over the current day", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_ads_percentage_today", help: "This represent the percentage of ads blocked over the current day", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_unique_domains", help: "This represent the number of unique domains seen", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_queries_forwarded", help: "This represent the number of queries forwarded", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_queries_cached", help: "This represent the number of queries cached", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_clients_ever_seen", help: "This represent the number of clients ever seen", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_unique_clients", help: "This represent the number of unique clients seen", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 New prometheus metric registered: Desc{fqName: "pihole_dns_queries_all_types", help: "This represent the number of DNS queries made for all types", constLabels: {}, variableLabels: []}
2019/05/09 09:32:19 Starting HTTP server
...
```

## Available options
```bash
# Interval of time the exporter will fetch data from PI-Hole
  -interval duration (optional) (default 10s)

# Hostname of the Raspberry PI where PI-Hole is installed
  -pihole_hostname string (optional) (default "127.0.0.1")

# Password defined on the PI-Hole interface
  -pihole_password string (optional)

# Port to be used for the exporter
  -port string (optional) (default "9311")
```
