# **2. MONITORING**

Collecting and monitoring metrics, statistics, and health statuses are important to keep systems sustainable and durable. There are lots of different tools for achieving this goal.


Also, we need to monitor and see how the inference tasks are distributed between inference worker instances, equally or not. We will observe this on Grafana.

In this project, we preferred [Prometheus](https://prometheus.io/) and [Telegraf](https://www.influxdata.com/time-series-platform/telegraf/) for collecting/scraping metrics, [InfluxDB](https://www.influxdata.com/) as storage database, and [Grafana](https://grafana.com/) for visualizing these data as information or charts in a dashboard.

<br>

**Monitoring topology:**

![Monitoring Topology](./images/04-monitoring-topology.drawio.svg)

<br>

## **2.1. Telegraf**

Should run in every individual host machine. Telegraf supports lots of different input and output plugins. In this project's architecture, it is configured as in [telegraf.conf](../monitoring/telegraf/telegraf.conf) file:

* Input (Docker container status): With ```[[inputs.docker]]``` input definition, the Telegraf agent communicates with the Docker Engine on the host machine via ```unix:///var/run/docker.sock``` socket. This ```docker.sock``` socket is mapped as a volume in [docker-compose.common.yml](../docker-compose.common.yml) file. With this input, Telegraf can collect various status and metrics data of Docker Engine.
<br>
For details of this plugin: [Docker Input Plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/docker).

* Input (HTTP listener for Prometheus Remote Write): With ```[[inputs.http_listener_v2]]``` input definition, the Telegraf agent listens at configured HTTP port ```1234``` for incoming data which in ```prometheusremotewrite``` format. Our Prometheus Agent instance will be writing its collected data to this endpoint.
<br>
For details of this plugin: [Prometheus Remote Write input data format](https://docs.influxdata.com/telegraf/v1.24/data_formats/input/prometheus-remote-write/) and [HTTP Listener v2 Input Plugin](https://github.com/influxdata/telegraf/tree/master/plugins/inputs/http_listener_v2).

* Output (InfluxDB v2): With ```[[outputs.influxdb_v2]]``` output definition, the Telegraf agent pushes its collected data (from Docker and Prometheus) to configured central InfluxDB database.
<br>
For details of this plugin: [InfluxDB v2.x Output Plugin](https://github.com/influxdata/telegraf/tree/master/plugins/outputs/influxdb_v2) and [Running InfluxDB 2.0 and Telegraf Using Docker](https://www.influxdata.com/blog/running-influxdb-2-0-and-telegraf-using-docker/).

<br>

## **2.2. Prometheus**

Should run in every individual host machine. Prometheus supports lots of different features and can be used for different aims. In this project's architecture, it is configured as in [prometheus.yml](../monitoring/prometheus/config/prometheus.yml) file.

* Despite it can be used as metrics storage database, in our architecture, we don't use its storage capabilities. We use it only to collect data, in agent mode, specifying ```'--enable-feature=agent``` argument in [docker-compose.common.yml](../docker-compose.common.yml) file. This configuration disables some features of Prometheus, like storage.
<br>
For details of Agent Mode: [Prometheus Agent Mode](https://prometheus.io/blog/2021/11/16/agent/#prometheus-agent-mode).

* It's not required to access web interface of Prometheus in our architecture, but it can be accessed on http://localhost:9000/prometheus via Nginx reverse proxy. To allow reverse proxying, ```--web.external-url``` and ```--web.route-prefix``` arguments are specified in [docker-compose.common.yml](../docker-compose.common.yml) file.
<br>
For details: [Configure Prometheus on a Sub-Path behind Reverse Proxy](https://blog.cubieserver.de/2020/configure-prometheus-on-a-sub-path-behind-reverse-proxy/).

* Input (Docker Service Discovery): We use Prometheus to let it discover services that can be replicated (in our case, only Inference service containers can be replicated), and scrape their metrics. To make Prometheus can discover our services, we configured its ```docker_sd_configs``` feature in [prometheus.yml](../monitoring/prometheus/config/prometheus.yml). Also, we configured some regular expressions to filter and transform container names.
<br>
Prometheus agent communicates with the Docker Engine on the host machine via ```unix:///var/run/docker.sock``` socket. This ```docker.sock``` socket is mapped as a volume in [docker-compose.common.yml](../docker-compose.common.yml) file. With this input, Prometheus can access list of running containers on Docker Engine.
<br>
For details: [Configuration of Service Discovery with docker_sd_config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#docker_sd_config), [Controlling the instance label](https://www.robustperception.io/controlling-the-instance-label/), and [Prometheus Relabel Rules and the ‘action’ Parameter
](https://blog.freshtracks.io/prometheus-relabel-rules-and-the-action-parameter-39c71959354a).

* Output (Telegraf): Prometheus will be pushing collected data to ```http://telegraf:1234/recieve``` as configured in [prometheus.yml](../monitoring/prometheus/config/prometheus.yml) file.
<br>
For details: [Configuration of remote_write](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write).

<br>

## **2.3. InfluxDB**

Should run in only the central host machine. InfluxDB is used as storage for collected metrics, it is configured by specifying environment variables starting with ```DOCKER_INFLUXDB_INIT``` in [docker-compose.common.yml](../docker-compose.common.yml) file and ```.env``` file.

<br>

## **2.4. Grafana**

Should run in only the central host machine. Grafana is used for visualizing these data as information or charts in a dashboard, it is configured by specifying environment variables in [docker-compose.common.yml](../docker-compose.common.yml) file and ```.env``` file, we didn't prefer to change and configure the ```grafana.ini```, Grafana allows us to configure it via environment variables.

Also, we use Grafana's Provisioning capabilities, which allows us to create initial datasources and dashboards. We mapped directories [monitoring/grafana/provisioning/](../monitoring/grafana/provisioning/) with ```/etc/grafana/provisioning``` in [docker-compose.common.yml](../docker-compose.common.yml) file.

It can be accessed on http://localhost:9000/grafana via Nginx reverse proxy. To allow reverse proxying, ```GF_SERVER_ROOT_URL``` and ```GF_SERVER_SERVE_FROM_SUB_PATH``` environment variables are specified in [docker-compose.common.yml](../docker-compose.common.yml) file.

For details: [Configuration Using environment variables](https://docs.huihoo.com/grafana/2.6/installation/configuration/index.html), and [Provision Grafana](https://grafana.com/docs/grafana/latest/administration/provisioning/).

<br>

## **2.5. Notes on mapping /var/run/docker.sock**

As we know, Docker Engine supports different host operating systems (e.g. Windows, Linux, MacOS, etc...). It can provide OS abstractions in most cases, but it can't in some cases, due to design differences between operating systems.

One of these differences is mapping ```/var/run/docker.sock``` UNIX socket path of Docker Engine. In our project, Telegraf and Prometheus instances should be allowed to access and have permission to this socket to communicate with Docker Engine.

For different host operating systems, the path mappings should be like this:

* **Linux:** ```/var/run/docker.sock:/var/run/docker.sock```
* **Windows:** ```//var/run/docker.sock:/var/run/docker.sock``` (an additional ```/``` character should exist as prefix, check out [here](https://stackoverflow.com/questions/36765138/bind-to-docker-socket-on-windows/41005007#41005007))
* **MacOS:** ```/var/run/docker.sock.raw:/var/run/docker.sock``` (the same mapping for Linux can be run as well, but it requires additional permission configurations in host MacOS, so found this trick [here](https://github.com/docker/for-mac/issues/4755))

To make our Docker Compose files supporting all of these host operating systems, we defined the mapping as:

```
    - ${DOCKER_SOCKET_PREFIX}/var/run/docker.sock${DOCKER_SOCKET_SUFFIX}:/var/run/docker.sock
```

And, we can manage this issue with environment variables, like:

* Adding an additional ```/``` character at start by specifying ```DOCKER_SOCKET_PREFIX="/"``` for Windows,
* Adding an additional ```.raw``` suffix by specifying ```DOCKER_SOCKET_SUFFIX=".raw"``` for MacOS,
* Or leave both of them empty as ```DOCKER_SOCKET_PREFIX=""``` and ```DOCKER_SOCKET_SUFFIX=""``` for Linux. Quotation marks should exist, because in ```.env``` file, these lines have some comments.

<br>

## **2.6. Notes on preferring Telegraf, Prometheus, and InfluxDB together**

Because of some best practices, because of some tools work *more compatible* together, or some of them were produced by same vendor, we name these toolsets as "stacks", like:

* **ELK Stack**: ElasticSearch, LogStash, Kibana
* **TICK Stack**: Telegraf, InfluxDB, Chronograf, and Kapacitor
* **TIG Stack**: Telegraf, InfluxDB, Grafana
* **Prometheus Stack**: Prometheus is different from other tools with its capabilities, I'm not sure there's a **stack** which is named, but it can be counted in this list.

<br>

### **Why not used Prometheus tools only?**

* At the first times of designing monitoring for this project, I thought to use Prometheus and Grafana, without Telegraf and InfluxDB. Because Prometheus has storage capability.

* We need a metric collector agent in each individual host, which should push its data to a central storage.
    * Telegraf is suitable for this, but it has no "service discovery" capability (as I know),
    * Prometheus (agent mode) can do service discovery and also push collected data to a central Prometheus storage, via Prometheus Remote Read/Remote Write. But at first times I'm not aware of its remote read/remote write capabilities.
    * But, after adopting Prometheus to our stack due to its service discovery capabilities, I encountered another handicap, that we need to use other tools like [cAdvisor](https://github.com/google/cadvisor), which has no official support for Apple Silicon (M1 chip), I need this support to run on my own computer :)
    * Then I have decided to separate metrics collection jobs into two pieces: Prometheus will be doing service discovery and collecting metrics of Inference instances via their ```:8000/metrics``` HTTP endpoints, then push them to Telegraf instance. Telegraf instance will be listening for an HTTP endpoint ```:1234/recieve``` for Prometheus, also collecting Docker container statistics, then pushing them to central InfluxDB instance.

<br>

## **2.7. Notes on Inference Service Metrics**

Inference service is written in Python, it uses [prometheus-client](https://pypi.org/project/prometheus-client/) library to expose ```:8000/metrics``` endpoint in Prometheus metrics format.

It exposes two types of metrics:
* **inference_time:** Time spent for each prediction task, in  [Summary metric type](https://prometheus.io/docs/concepts/metric_types/#summary),
* **inference_frame_count:** A counter which is increased for each done prediction, in [Counter metric type](https://prometheus.io/docs/concepts/metric_types/#counter). This counter value always increases. To calculate processed frame count per second approximately, at Grafana side, we configured using [NON_NEGATIVE_DERIVATIVE()](https://archive.docs.influxdata.com/influxdb/v0.13/query_language/functions/#non-negative-derivative) function.

<br>

---

<div align="right">

[&lt;&nbsp;&nbsp;Previous chapter: APPLICATION](./01-APPLICATION.md)&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;|&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;[Next chapter: CONCLUSION&nbsp;&nbsp;&gt;](./03-CONCLUSION.md)

</div>