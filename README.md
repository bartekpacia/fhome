# fhome

Package and CLI to communicate with [F&Home – a smart home
system](https://www.fhome.pl).

> Note: extremely alpha, work in progress

F&Home doesn't provide any kind of API, but I managed to figure out how it works
using Chrome Devtools and looking at the messages it sends over websockets.

## Using CLI

First, you have to build it:

```
$ make
```

Then just builting help:

```
./fh --help
```

## Using package

There is `fhome` package that the cli (in `cmd`) is using. You can use it
yourself, bu
