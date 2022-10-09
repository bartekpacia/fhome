# fhome

[![Go Reference][go-reference-badge]][go-reference-link] [![Go
Report][go-report-badge]][go-report-link]

Package and CLI to communicate with [F&Home – a smart home system][fhome].

F&Home doesn't provide any kind of API, but I managed to figure out how it works
using Chrome Devtools and by looking at the messages it sends over websockets.

Then I started putting together this project.

## Packages

This project consists of several packages.

### fhome

Core package implementing the F&Home API. Use it if you want make your own
program interacting with it.

### fh

Command-line program to easily interact with your F&Home-enabled devices.

Depends on the `fhome` package.

**Build**

```console
$ go build -o fh ./cmd/fh/*.go
```

**Install**

```console
$ go install ./cmd/fh
```

**Help**

```console
$ fh help
```

### fhomed

Provides integration between F&Home and HomeKit. Intended to be used as a
background daemon.

Depends on the `fhome` package.

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
