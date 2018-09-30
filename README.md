# kubernetes-asaka-plugin

This is a repo for the device plugins of Asaka to integrate with Kubernetes v1.12.0. And it's referenced to [NVIDIA device plugin for Kubernetes](https://github.com/NVIDIA/k8s-device-plugin) and original integration between Asaka and Kubernetes.

## Prerequisite

Install [Golang](https://golang.org/dl) with version 1.9 or later

## How to build

```bash
git clone https://eos2git.cec.lab.emc.com/OCTO/kubernetes-asaka-plugin.git
cd kubernetes-asaka-plugin
GOPATH=`pwd` make compile
```

## How to Run

```bash
LOG_LEVEL=info XAAS_CONTROLLER_URI=127.0.0.1:9527 bin/asaka-vgpu
```
