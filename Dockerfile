ARG PLATFORM=linux/arm
FROM --platform=${PLATFORM} python:3.7-alpine
MAINTAINER Christophe Lambin <christophe.lambin@gmail.com>

RUN mkdir /app
WORKDIR /app

COPY *.py Pip* /app/

RUN pip install --upgrade pip && \
    pip install pipenv && \
    pipenv install --dev --system --deploy --ignore-pipfile

EXPOSE 8080

CMD python pinger.py
