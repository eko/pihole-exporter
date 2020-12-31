# PI-Hole Prometheus Exporter

![Build/Push (master)](https://github.com/eko/pihole-exporter/workflows/Build/Push%20(master)/badge.svg)
[![GoDoc](https://godoc.org/github.com/eko/pihole-exporter?status.png)](https://godoc.org/github.com/eko/pihole-exporter)
[![GoReportCard](https://goreportcard.com/badge/github.com/eko/pihole-exporter)](https://goreportcard.com/report/github.com/eko/pihole-exporter)

This is a Prometheus exporter for [PI-Hole](https://pi-hole.net/)'s Raspberry PI ad blocker.

![Grafana dashboard](https://raw.githubusercontent.com/eko/pihole-exporter/master/dashboard.jpg)

Grafana dashboard is [available here](https://grafana.com/dashboards/10176) on the Grafana dashboard website and also [here](https://raw.githubusercontent.com/eko/pihole-exporter/master/grafana/dashboard.json) on the GitHub repository.

## Prerequisites

* [Go](https://golang.org/doc/)

## Installation

### Download binary

You can download the latest version of the binary built for your architecture here:

* Architecture **i386** [
    [Linux](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-linux-386) /
    [Windows](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-windows-386.exe)
]
* Architecture **amd64** [
    [Darwin](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-darwin-amd64) /
    [Linux](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-linux-amd64) /
    [Windows](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-windows-amd64.exe)
]
* Architecture **arm** [
    [Darwin](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-darwin-arm64) /
    [Linux](https://github.com/eko/pihole-exporter/releases/latest/download/pihole_exporter-linux-arm)
]

### Using Docker

The exporter is also available as a [Docker image](https://hub.docker.com/r/ekofr/pihole-exporter).
You can run it using the following example and pass configuration environment variables:

```
$ docker run \
  -e 'PIHOLE_HOSTNAME=192.168.1.2' \
  -e 'PIHOLE_PASSWORD=mypassword' \
  -e 'INTERVAL=30s' \
  -e 'PORT=9617' \
  -p 9617:9617 \
  ekofr/pihole-exporter:latest
```

Or use PiHole's `WEBPASSWORD` as an API token instead of the password

```bash
$ API_TOKEN=$(awk -F= -v key="WEBPASSWORD" '$1==key {print $2}' /etc/pihole/setupVars.conf)
$ docker run \
  -e 'PIHOLE_HOSTNAME=192.168.1.2' \
  -e "PIHOLE_API_TOKEN=$API_TOKEN" \
  -e 'INTERVAL=30s' \
  -e 'PORT=9617' \
  ekofr/pihole-exporter:latest
```

If you are running pi-hole behind https, you must both set the `PIHOLE_PROTOCOL` environment variable
as well as include your ssl certificates to the docker image as it does not have any baked in:

```
$ docker run \
  -e 'PIHOLE_PROTOCOL=https' \
  -e 'PIHOLE_HOSTNAME=192.168.1.2' \
  -e 'PIHOLE_PASSWORD=mypassword' \
  -e 'INTERVAL=30s' \
  -e 'PORT=9617' \
  -v '/etc/ssl/certs:/etc/ssl/certs:ro' \
  -p 9617:9617 \
  ekofr/pihole-exporter:latest
```

### From sources

Optionally, you can download and build it from the sources. You have to retrieve the project sources by using one of the following way:
```bash
$ go get -u github.com/eko/pihole-exporter
# or
$ git clone https://github.com/eko/pihole-exporter.git
```

Install the needed vendors:

```
$ GO111MODULE=on go mod vendor
```

Then, build the binary (here, an example to run on Raspberry PI ARM architecture):
```bash
$ GOOS=linux GOARCH=arm GOARM=7 go build -o pihole_exporter .
```

## Usage

In order to run the exporter, type the following command (arguments are optional):

Using a password

```bash
$ ./pihole_exporter -pihole_hostname 192.168.1.10 -pihole_password azerty
```

Or use PiHole's `WEBPASSWORD` as an API token instead of the password

```bash
$ API_TOKEN=$(awk -F= -v key="WEBPASSWORD" '$1==key {print $2}' /etc/pihole/setupVars.conf)
$ ./pihole_exporter -pihole_hostname 192.168.1.10 -pihole_api_token $API_TOKEN
```

```bash
2019/05/09 20:19:52 ------------------------------------
2019/05/09 20:19:52 -  PI-Hole exporter configuration  -
2019/05/09 20:19:52 ------------------------------------
2019/05/09 20:19:52 PIHoleHostname : 192.168.1.10
2019/05/09 20:19:52 PIHolePassword : azerty
2019/05/09 20:19:52 Port : 9617
2019/05/09 20:19:52 Interval : 10s
2019/05/09 20:19:52 ------------------------------------
2019/05/09 20:19:52 New Prometheus metric registered: domains_blocked
2019/05/09 20:19:52 New Prometheus metric registered: dns_queries_today
2019/05/09 20:19:52 New Prometheus metric registered: ads_blocked_today
2019/05/09 20:19:52 New Prometheus metric registered: ads_percentag_today
2019/05/09 20:19:52 New Prometheus metric registered: unique_domains
2019/05/09 20:19:52 New Prometheus metric registered: queries_forwarded
2019/05/09 20:19:52 New Prometheus metric registered: queries_cached
2019/05/09 20:19:52 New Prometheus metric registered: clients_ever_seen
2019/05/09 20:19:52 New Prometheus metric registered: unique_clients
2019/05/09 20:19:52 New Prometheus metric registered: dns_queries_all_types
2019/05/09 20:19:52 New Prometheus metric registered: reply
2019/05/09 20:19:52 New Prometheus metric registered: top_queries
2019/05/09 20:19:52 New Prometheus metric registered: top_ads
2019/05/09 20:19:52 New Prometheus metric registered: top_sources
2019/05/09 20:19:52 New Prometheus metric registered: forward_destinations
2019/05/09 20:19:52 New Prometheus metric registered: querytypes
2019/05/09 20:19:52 New Prometheus metric registered: status
2019/05/09 20:19:52 Starting HTTP server
2019/05/09 20:19:54 New tick of statistics: 648 ads blocked / 66796 total DNS querie
...
```

Once the exporter is running, you also have to update your `prometheus.yml` configuration to let it scrape the exporter:

```yaml
scrape_configs:
  - job_name: 'pihole'
    static_configs:
      - targets: ['localhost:9617']
```

## Available CLI options
```bash
# Interval of time the exporter will fetch data from PI-Hole
  -interval duration (optional) (default 10s)

# Hostname of the Raspberry PI where PI-Hole is installed
  -pihole_hostname string (optional) (default "127.0.0.1")

# Password defined on the PI-Hole interface
  -pihole_password string (optional)


# WEBPASSWORD / api token defined on the PI-Hole interface at `/etc/pihole/setupVars.conf`
  -pihole_api_token string (optional)

# Port to be used for the exporter
  -port string (optional) (default "9617")
```

## Available Prometheus metrics

| Metric name                  | Description                                                                               |
|:----------------------------:|-------------------------------------------------------------------------------------------|
| pihole_domains_being_blocked | This represent the number of domains being blocked                                        |
| pihole_dns_queries_today     | This represent the number of DNS queries made over the current day                        |
| pihole_ads_blocked_today     | This represent the number of ads blocked over the current day                             |
| pihole_ads_percentage_today  | This represent the percentage of ads blocked over the current day                         |
| pihole_unique_domains        | This represent the number of unique domains seen                                          |
| pihole_queries_forwarded     | This represent the number of queries forwarded                                            |
| pihole_queries_cached        | This represent the number of queries cached                                               |
| pihole_clients_ever_seen     | This represent the number of clients ever seen                                            |
| pihole_unique_clients        | This represent the number of unique clients seen                                          |
| pihole_dns_queries_all_types | This represent the number of DNS queries made for all types                               |
| pihole_reply                 | This represent the number of replies made for all types                                   |
| pihole_top_queries           | This represent the number of top queries made by PI-Hole by domain                        |
| pihole_top_ads               | This represent the number of top ads made by PI-Hole by domain                            |
| pihole_top_sources           | This represent the number of top sources requests made by PI-Hole by source host          |
| pihole_forward_destinations  | This represent the number of forward destinations requests made by PI-Hole by destination |
| pihole_querytypes            | This represent the number of queries made by PI-Hole by type                              |
| pihole_status                | This represent if PI-Hole is enabled                                                      |


## Pihole-Exporter Helm Chart

[Link](https://github.com/SiM22/pihole-exporter-helm-chart)

This is a simple Helm Chart to deploy the exporter in a kubernetes cluster.
