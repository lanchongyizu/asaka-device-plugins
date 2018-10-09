# asaka-device-plugins

This is a repo for the device plugins of Asaka to integrate with Kubernetes v1.12.0. And it's referenced to [NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin) and original integration between Asaka and Kubernetes.

## Prerequisite

Install [Golang](https://golang.org/dl) with version 1.9 or later

## How to build

```bash
git clone https://github.com/lanchongyizu/asaka-device-plugins
cd asaka-device-plugins
GOPATH=`pwd` make compile
```

## How to Run

```bash
XAAS_CONTROLLER_URI=127.0.0.1:9527 bin/asaka-vgpu
```
