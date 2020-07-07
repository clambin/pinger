# pinger
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/clambin/pinger?color=green&label=Release&style=plastic)
![Codecov](https://img.shields.io/codecov/c/gh/clambin/pinger?style=plastic)
![Gitlab pipeline status (branch)](https://img.shields.io/gitlab/pipeline/clambin/pinger/master?style=plastic)
![GitHub](https://img.shields.io/github/license/clambin/pinger?style=plastic)

Born on a rainy Sunday afternoon, when my ISP was being its unreliable self again.  Measures the latency and packet loss to one of more hosts and reports the data to Prometheus.

## Getting started

### Docker

Pinger can be installed in a Docker container via docker-compose:

```
version: '2'
services:
    pinger:
        image: clambin/pinger:latest
        container_name: pinger
        command: --interval 5 
        environment:
            - HOSTS=192.168.0.1
        ports:
            - 8080:8080/tcp
        restart: unless-stopped
```

### Metrics

Pinger exposes the following metrics to Prometheus:

```
* pinger_latency:     Average latency measured over the last interval
* pinger_packet_loss: Total packet loss measured over the last interval
```

### Command line arguments:

The following command line arguments can be passed to pimon:

```
usage: pinger.py [-h] [--version] [--interval INTERVAL] [--once] [--debug]
                 [--port PORT]
                 [host [host ...]]

positional arguments:
  host                 Target host / IP address

optional arguments:
  -h, --help           show this help message and exit
  --version            show program's version number and exit
  --interval INTERVAL  Time between measurements (default: 5 sec)
  --once               Measure once and then terminate
  --debug              Set logging level to debug
  --port PORT          Prometheus port (default: 8080)
```

The target hosts can also be provided by exporting an environment variable 'HOSTS'. If both are provided, the environment variable takes precedence.

## Authors

* **Christophe Lambin**

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
