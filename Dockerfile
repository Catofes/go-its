FROM ubuntu:16.04

RUN apt update && apt install -y ca-certificates

RUN mkdir /usr/app

COPY . /usr/app

WORKDIR /usr/app

CMD ["/usr/app/server","-conf","/usr/app/server.json"]
