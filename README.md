> Welcome to F&Home – the worst smart home system ever.
>
> – me

> I did this thing not because it's easy, but because I thought it would be easy.

# fhome

[![Go Reference][go-reference-badge]][go-reference-link]
[![Go Report][go-report-badge]][go-report-link]

Package and CLI to communicate with [F&Home – a smart home system][fhome].

F&Home doesn't provide any kind of API, but I managed to figure out how it works
using Chrome Devtools and by looking at the messages it sends over websockets.

Then I started putting together this project.

## Package

The `api` package implements the F&Home API.
Use it if you want to make your own program interact with it.

## Command-line apps

Both `fhome`, `fhome-homekit`, `fhome-web`, and `fhome-exporter` read configuration from TOML files.
Config files are loaded in order (later overrides earlier), and environment variables override file values.

**Config file locations**

- `/etc/fhome/config.toml`
- `~/.config/fhome/config.toml`

(these locations are common for all CLI apps)

**Required keys**

| Key                       | Description                |
| --------------------------| ---------------------------|
| `FHOME_EMAIL`             | F&Home account email       |
| `FHOME_CLOUD_PASSWORD`    | F&Home cloud password      |
| `FHOME_RESOURCE_PASSWORD` | Resource (device) password |

**Example config**

```toml
FHOME_EMAIL = "you@example.com"
FHOME_CLOUD_PASSWORD = "your-cloud-password"
FHOME_RESOURCE_PASSWORD = "your-resource-password"
```

### fhome

Command-line program to easily interact with your F&Home-enabled devices.

Depends on the `api` package.

**Build**

```console
$ go build -o fhome ./cmd/fhome/*.go
```

**Install**

```console
$ go install ./cmd/fhome
```

**Help**

```console
$ fhome help
```

### fhome-homekit

HomeKit bridge for F&Home.
Provides bidirectional sync between F&Home devices and Apple HomeKit.
Intended to be used as a background daemon.

Depends on the `api` package.

**Build**

```console
$ go build -o fhome-homekit ./cmd/fhome-homekit
```

**Install**

```console
$ go install ./cmd/fhome-homekit
```

**Register with systemd**

1. Copy the binary to a common location

    ```console
    $ sudo cp ./fhome-homekit /usr/local/bin
    ```

2. Create a service file

    ```console
    $ sudo cp ./fhome-homekit.service /etc/systemd/system
    ```

3. Reload changes

    ```console
    $ sudo systemctl daemon-reload
    ```

**Extract status logs from journald**

```console
$ journalctl \
  _SYSTEMD_UNIT=fhome-homekit.service \
  --no-pager \
  --output json-pretty \
  | jq --slurp \
    --compact-output \
     '.[] | {timestamp: .__REALTIME_TIMESTAMP, msg: .MESSAGE}'
```

Or in a single line:

```console
$ journalctl _SYSTEMD_UNIT=fhome-homekit.service --no-pager -o json-pretty | jq -s -c  '.[] | {timestamp: .__REALTIME_TIMESTAMP, msg: .MESSAGE}'
```

### fhome-exporter

Prometheus exporter that exposes F&Home temperature sensor data at `/metrics`.

Reads the same config files as the other CLI apps.

**Metrics example**

```
# HELP fhome_room_temperature_celsius Room temperature in degrees Celsius
# TYPE fhome_room_temperature_celsius gauge
fhome_room_temperature_celsius{panel="Ogrzewanie",room="Bartek",cell_id="439"} 22.5
```

**Build**

```console
$ go build -o fhome-exporter ./cmd/fhome-exporter
```

**Install**

```console
$ go install ./cmd/fhome-exporter
```

**Docker**

```console
$ docker build -t fhome-exporter -f cmd/fhome-exporter/Dockerfile .
$ docker run -p 9222:9222 -v ~/.config/fhome:/root/.config/fhome:ro fhome-exporter
```

**Flags**

| Flag      | Default | Description                  |
| --------- | ------- | ---------------------------- |
| `--port`  | `9222`  | Port to listen on            |
| `--json`  |         | Output logs in JSON Lines    |
| `--debug` |         | Show debug logs              |

### fhome-web

A (currently dummy) web server for F&Home device preview.
Provides a simple web UI for viewing devices and a `/gate` endpoint for quick device control.

Depends on the `api` package.

**Build**

```console
$ go build -o fhome-web ./cmd/fhome-web
```

**Install**

```console
$ go install ./cmd/fhome-web
```

[go-reference-badge]: https://pkg.go.dev/badge/github.com/bartekpacia/fhome.svg
[go-reference-link]: https://pkg.go.dev/github.com/bartekpacia/fhome
[go-report-badge]: https://goreportcard.com/badge/github.com/bartekpacia/fhome
[go-report-link]: https://goreportcard.com/report/github.com/bartekpacia/fhome
[fhome]: https://www.fhome.pl
