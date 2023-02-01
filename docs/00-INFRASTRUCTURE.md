# **0. INFRASTRUCTURE**

This project offers different types of topologies which are described in main [README](../README.md#installation-and-running) file.

In this documentation, we used a configuration consisting of a MacBook Pro M1 Chip and a desktop machine with Nvidia GPUs.

<br>

**Application topology:**

![Pipeline Diagram](./images/01-pipeline.drawio.svg)

<br>

## **0.1. EXAMPLE CONFIGURATION (Single-host)**

If you want to try this project on only one host machine, you can choose the proper profile option:

* Single-host without GPU (CUDA), with CPU:

It's not required to edit ```.env``` file, leave it as default.

Execute command:
```sh
$ docker-compose --profile single_host_cpu up -d
```

* Single-host with GPU (CUDA):

```.env``` file should be edited as:

```
...
CUDA_VERSION={version of CUDA library installed on the host machine} # e.g. 11.4.2
CUDNN_VERSION={version of CuDNN library installed on the host machine} # e.g. 8
...
```

Execute command:
```sh
$ docker-compose --profile single_host_gpu up -d
```

<br>

## **0.2. EXAMPLE CONFIGURATION (My configuration, Multi-host)**

### **a. MacBook Pro:**

Used as "central host", with inference workers.

My MacBook Pro machine is Apple Silicon M1 (ARM64/AARCH64) version. Inference workers will run in CPU mode.

* ```.env``` file was edited as:

```
HOSTNAME_TAG=mbp
DOCKER_SOCKET_PREFIX="" # should be "/" if host operating system is Windows. If Linux or MacOS, should be left blank "".
DOCKER_SOCKET_SUFFIX=".raw" # should be ".raw" if host operating system is MacOS. If Linux or Windows, should be left blank "".

WEB_DOMAIN=localhost
WEB_HTTP_PORT=9000
REDIS_HOST=redis # should be "redis" if single host configuration, e.g. 192.168.0.15 in distributed configuration
REDIS_PORT=6379
REDIS_PASSWORD=922Y2ZFgZzGpPpZj
MEDIABRIDGE_DOCKER_HOST_IP="" # e.g. 192.168.0.11
MEDIABRIDGE_UDP_PORT=12000

CUDA_VERSION="" # e.g. 11.4.2
CUDNN_VERSION="" # e.g. 8

DOCKER_INFLUXDB_INIT_USERNAME=influxuser
DOCKER_INFLUXDB_INIT_PASSWORD=LJYXounx0jPE69swDebv
DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=NkLY6Z3Et556rCLjVcrh
DOCKER_INFLUXDB_INIT_ORG=adalkiran
DOCKER_INFLUXDB_INIT_BUCKET=distributed-inference

INFLUXDB_HOST=influxdb # should be "influxdb" if single host configuration, e.g. 192.168.0.15 in distributed configuration
INFLUXDB_PORT=8086

GF_SECURITY_ADMIN_PASSWORD=admin
```

* Ports 6379 (Redis) and 8086 (InfluxDB) should be made allowed on the firewall. Or Docker Engine should be allowed on the firewall.

* Containers were configured with command, using ```central_with_inference_cpu``` profile:

```sh
$ docker-compose --profile central_with_inference_cpu up -d
```

<br>

### **b. Desktop:**

Used as "only inference with GPU host".

My desktop machine has Nvidia GPUs. Ubuntu 22.04.1 LTS is running. CUDA 11.4.2 and CuDNN 8 were already installed.

* ```.env``` file was edited as:

```
HOSTNAME_TAG=desktop
DOCKER_SOCKET_PREFIX="" # should be "/" if host operating system is Windows. If Linux or MacOS, should be left blank "".
DOCKER_SOCKET_SUFFIX="" # should be ".raw" if host operating system is MacOS. If Linux or Windows, should be left blank "".

WEB_DOMAIN=localhost
WEB_HTTP_PORT=9000
REDIS_HOST={ip_of_central_host like 192.168.X.X} # should be "redis" if single host configuration, e.g. 192.168.0.15 in distributed configuration
REDIS_PORT=6379
REDIS_PASSWORD=922Y2ZFgZzGpPpZj
MEDIABRIDGE_DOCKER_HOST_IP="" # e.g. 192.168.0.11
MEDIABRIDGE_UDP_PORT=12000

CUDA_VERSION="11.4.2" # e.g. 11.4.2
CUDNN_VERSION="8" # e.g. 8

DOCKER_INFLUXDB_INIT_USERNAME=influxuser
DOCKER_INFLUXDB_INIT_PASSWORD=LJYXounx0jPE69swDebv
DOCKER_INFLUXDB_INIT_ADMIN_TOKEN=NkLY6Z3Et556rCLjVcrh
DOCKER_INFLUXDB_INIT_ORG=adalkiran
DOCKER_INFLUXDB_INIT_BUCKET=distributed-inference

INFLUXDB_HOST={ip_of_central_host like 192.168.X.X} # should be "influxdb" if single host configuration, e.g. 192.168.0.15 in distributed configuration
INFLUXDB_PORT=8086

GF_SECURITY_ADMIN_PASSWORD=admin
```

* Containers were configured with command, using ```inference_gpu``` profile:

```sh
$ docker-compose --profile inference_gpu up -d
```

<br>

---

<div align="right">

[&lt;&nbsp;&nbsp;Documentation Index](./README.md)&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Next chapter: APPLICATION&nbsp;&nbsp;&gt;](./01-APPLICATION.md)

</div>