import argparse
import os

import version


def str2bool(v):
    if isinstance(v, bool):
        return v
    if v.lower() in ('yes', 'true', 't', 'y', '1', 'on'):
        return True
    elif v.lower() in ('no', 'false', 'f', 'n', '0', 'off'):
        return False
    else:
        raise argparse.ArgumentTypeError('Boolean value expected.')


def get_configuration(args=None):
    default_interval = 5
    default_port = 8080
    default_host = ['127.0.0.1']
    default_log = 'logfile.csv'

    parser = argparse.ArgumentParser()
    parser.add_argument('--version', action='version', version=f'%(prog)s {version.version}')
    parser.add_argument('--interval', type=int, default=default_interval,
                        help=f'Time between measurements (default: {default_interval} sec)')
    parser.add_argument('--once', action='store_true',
                        help='Measure once and then terminate')
    parser.add_argument('--debug', action='store_true',
                        help='Set logging level to debug')
    # Reporters
    parser.add_argument('--reporter-prometheus', type=str2bool, nargs='?', default=True,
                        help='Report metrics to Prometheus')
    parser.add_argument('--port', type=int, default=default_port,
                        help=f'Prometheus port (default: {default_port})')
    parser.add_argument('--reporter-logfile', type=str2bool, nargs='?', default=False,
                        help='Report metrics to a CSV logfile')
    parser.add_argument('--logfile', action='store', default=default_log,
                        help=f'metrics output logfile (default: {default_log})')
    # Hosts to ping
    parser.add_argument('hosts', nargs='*', default=default_host, metavar='host',
                        help='Target host / IP address')
    args = parser.parse_args(args)
    # env var HOSTS overrides commandline args
    if 'HOSTS' in os.environ:
        args.hosts = os.environ.get('HOSTS').split()
    return args


def print_configuration(config):
    return ', '.join([f'{key}={val}' for key, val in vars(config).items()])
