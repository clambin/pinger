FROM python:3.7-alpine
MAINTAINER Christophe Lambin <christophe.lambin@gmail.com>

RUN mkdir /app
WORKDIR /app

COPY *.py Pip* ./
COPY metrics/*.py metrics/

RUN pip install --upgrade pip && \
    pip install pipenv && \
    pipenv install --dev --system --deploy --ignore-pipfile

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/python3", "pinger.py"]
CMD []
