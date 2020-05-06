from libpinger.pinger import pinger
from libpinger.configuration import get_configuration

if __name__ == '__main__':
    pinger(get_configuration())
