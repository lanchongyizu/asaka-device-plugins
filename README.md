# What's this

This is the device plugins for Asaka.


# How to build

## Prerequisite

Install [Golang](https://golang.org/dl) with version 1.9 or later and [direnv](https://github.com/direnv/direnv) to set `GOPATH`

## Compile

Clone repository.
```bash
git clone https://github.com/lanchongyizu/asaka-device-plugins
cd asaka-device-plugins
direnv allow .
```

Build asaka-vgpu:
```bash
GOPATH=`pwd` make compile
```

