# Epiphaneia

A "simple" command-line DMM GAMES PLAYER launcher written in Go.

## :warning: Notice :warning:

You can only run games that you actually own on
your DMM GAMES account.
You cannot add new free games or buy games from
the library, and must be done through the official
website or client.

## How to use

While this project works under Linux, this is currently meant to be run under
a Wine prefix for now. To cross-compile:

```
GOOS=windows GOARCH=amd64 go build -v
```

`Epiphaneia.exe -help` should contain most of the information you need.

## Future plans

* Support for Linux (Wine/Proton integration)
* FANZA GAMES support (I will need a bottle of bleach for this)

## License

Licensed under BSD 3-clause

## Disclaimer

Not affiliated with DMM.com. Use at your own risk.
