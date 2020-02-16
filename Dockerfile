ARG BASE_IMAGE= python:3.7-alpine
FROM $BASE_IMAGE
MAINTAINER Christophe Lambin <christophe.lambin@gmail.com>

RUN mkdir /app
WORKDIR /app

COPY *.py Pip* ./
COPY metrics/*.py metrics/

RUN apk add iputils && \
    pip install --upgrade pip && \
    pip install pipenv && \
    pipenv install --dev --system --deploy --ignore-pipfile

EXPOSE 8080

RUN addgroup -S -g 1000 abc && adduser -S --uid 1000 --ingroup abc abc
USER abc

ENTRYPOINT ["/usr/local/bin/python3", "pinger.py"]
CMD []
