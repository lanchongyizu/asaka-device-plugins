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

## Test Results of Release Interface

Release is added in the KillPod function of Kubelet, and the test results are as follows by `kubectl delete pod <pod_name>`.

1. long run application

Released once

2. non long run application with exit 0

| restart policy | release times |
| :------------- | :-----------: |
| Always         | 0             |
| OnFailure      | 0             |
| Never          | 0             |

3. application with exit 1

| restart policy | release times |
| :------------- | :-----------: |
| Always         | 1             |
| OnFailure      | 1             |
| Never          | 1             |

4. container doesn't start

image not found, and fail to pull image

| restart policy | release times |
| :------------- | :-----------: |
| Always         | 0             |
| OnFailure      | 1             |
| Never          | 0             |
