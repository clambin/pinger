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
usage: pinger [<flags>] [<hosts>...]

pinger

Flags:
  -h, --help                 Show context-sensitive help (also try --help-long and --help-man).
  -v, --version              Show application version.
      --port=8080            Metrics listener port
      --debug                Log debug messages

Args:
  [<hosts>]  hosts to ping

```

The target hosts can also be provided by exporting an environment variable 'HOSTS', e.g.

```
export HOSTS="127.0.0.1 192.168.0.1 192.168.0.200"
```

If both are provided, the environment variable takes precedence.

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
