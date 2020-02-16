# pinger

Born on a rainy Sunday afternoon, when my ISP was being its unreliable self again.  Measures the latency and packet loss to one of more hosts and reports the data to Prometheus.

## Getting started

### Docker

Pinger can be installed in a Docker container via docker-compose:

```
version: '2'
services:
    pinger:
        image: clambin/pinger
        container_name: pinger
        environment:
            - HOSTS=192.168.0.1
        ports:
            - 8080:8080/tcp
        restart: unless-stopped
```

### Metrics

Pinger exposes the following metrics to Prometheus:

```
* pinger_latency:  Average latency measured over the last interval
* ping_packetloss: Total packet loss measured over the last interval
```

### Command line arguments:

The following command line arguments can be passed to pimon:

```
usage: pinger.py [-h] [--version] [--interval INTERVAL] [--port PORT]
                 [--debug]
                 [host [host ...]]

positional arguments:
  host                 Target host / IP address

optional arguments:
  -h, --help           show this help message and exit
  --version            show program's version number and exit
  --interval INTERVAL  Time between measurements (default: 60 sec)
  --port PORT          Prometheus port (default: 8080)
  --debug              Set logging level to debug
```

The target hosts can also be provided by exporting an environment variable 'HOSTS'. If both are provided, the environment variable takes precedence.

## Authors

* **Christophe Lambin**

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.


