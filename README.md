# pinger
![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/clambin/pinger?color=green&label=Release&style=plastic)
![Build](https://github.com/clambin/pinger/workflows/Build/badge.svg)
![Codecov](https://img.shields.io/codecov/c/gh/clambin/pinger?style=plastic)
![Go Report Card](https://goreportcard.com/badge/github.com/clambin/pinger)
![GitHub](https://img.shields.io/github/license/clambin/pinger?style=plastic)

Born on a rainy Sunday afternoon, when my ISP was being its unreliable self again.  Measures the latency and packet loss to one of more hosts and reports the data to Prometheus.

## Getting started
### Command line arguments:

The following command line arguments can be passed:

```
Usage:
  pinger [flags] [ <host> ... ]

Flags:
      --addr string     Metrics listener address (default ":8080")
      --config string   Configuration file
      --debug           Log debug messages
  -h, --help            help for pinger
      --port int        Metrics listener port (obsolete)
  -v, --version         version for pinger
```

### Configuration file
The configuration file option specifies a yaml-formatted configuration file::

```
# Log debug messages
debug: true
# Metrics listener address (default ":8080")
addr: :8080
# Targets to ping
targets: 
  - host: 127.0.0.1  # Host IP address of hostname (mandatory)
    name: localhost  # Name to use for prometheus metrics (optional; pinger uses host if name is not specified)
```

If the filename is not specified on the command line, pinger will look for a file `config.yaml` in the following directories:

```
/etc/pinger
$HOME/.pinger
.
```

Any value in the configuration file may be overriden by setting an environment variable with a prefix `PINGER_`.


The target hosts can also be provided by exporting an environment variable 'HOSTS', e.g.

```
export HOSTS="127.0.0.1 192.168.0.1 192.168.0.200"
```

Pinger will consider provided hosts in the following order:

- HOSTS environment variable
- command-line arguments
- configuration file

NOTE: support for the HOSTS environment variable and command-line arguments will be removed in a future release. 

### Docker

Images for arm, arm64 & amd64 are available on [ghcr.io](https://ghcr.io/clambin/pinger).

### Metrics

Pinger exposes the following metrics to Prometheus:

```
* pinger_packet_count:         Total packets sent
* pinger_packet_loss_count:    Total packet loss measured 
* pinger_latency_seconds:      Total latency measured
```

## Authors

* **Christophe Lambin**

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
