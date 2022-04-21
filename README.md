# fhome

Package and CLI to communicate with [F&Home – a smart home
system](https://www.fhome.pl).

F&Home doesn't provide any kind of API, but I managed to figure out how it works
using Chrome Devtools and by looking at the messages it sends over websockets.

Then I started putting together this project.

## Using CLI

First, you have to build it:

```
$ make
```

Then just see help:

```
./fh --help
```

## Using package

There is `fhome` package that the CLI (in `cmd`) is using. It is independent
from the CLI.
