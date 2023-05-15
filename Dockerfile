FROM golang:1.11.4

RUN mkdir -p /go/src/app

COPY bin /go/src/app
WORKDIR /go/src/app

CMD ["/go/src/app/server"]
