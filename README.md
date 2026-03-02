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

Both `fhome` and `fhomed` read configuration from TOML files.
Config files are loaded in order (later overrides earlier), and environment variables override file values.

**Config file locations:**

- `fhome`: `/etc/fhome/config.toml`, `~/.config/fhome/config.toml`
- `fhomed`: `/etc/fhomed/config.toml`, `~/.config/fhomed/config.toml`

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

### fhomed

Provides integration between F&Home and HomeKit. Intended to be used as a
background daemon.

Depends on the `api` package.

**Registering with systemd**

1. Copy the binary to a common location

    ```console
    $ sudo cp ./fhomed /usr/local/bin
    ```

2. Create a service file

    ```console
    $ sudo cp ./fhomed.service /etc/systemd/system
    ```

3. Reload changes

    ```console
    $ sudo systemctl daemon-reload
    ```

**Extracting status logs from journald**

```console
$ journalctl \
  _SYSTEMD_UNIT=fhomed.service \
  --no-pager \
  --output json-pretty \
  | jq --slurp \
    --compact-output \
     '.[] | {timestamp: .__REALTIME_TIMESTAMP, msg: .MESSAGE}'
```

Or in a single line:

```console
$ journalctl _SYSTEMD_UNIT=fhomed.service --no-pager -o json-pretty | jq -s -c  '.[] | {timestamp: .__REALTIME_TIMESTAMP, msg: .MESSAGE}'
```

**Build**

```console
$ go build -o fhomed ./cmd/fhomed/*.go
```

**Install**

```console
$ go install ./cmd/fhomed
```

[go-reference-badge]: https://pkg.go.dev/badge/github.com/bartekpacia/fhome.svg

[go-reference-link]: https://pkg.go.dev/github.com/bartekpacia/fhome

[go-report-badge]: https://goreportcard.com/badge/github.com/bartekpacia/fhome

[go-report-link]: https://goreportcard.com/report/github.com/bartekpacia/fhome

[fhome]: https://www.fhome.pl
