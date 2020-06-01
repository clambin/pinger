from src.pinger import pinger
from src.configuration import get_configuration

if __name__ == '__main__':
    pinger(get_configuration())
