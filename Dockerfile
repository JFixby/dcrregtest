FROM golang:1.11

WORKDIR /go/src/github.com/jfixby/dcrregtest
COPY . .

RUN apt-get update && apt-get upgrade -y && apt-get install -y rsync

RUN git clone -b release-v1.4 https://github.com/decred/dcrd /go/src/github.com/decred/dcrd

RUN cd /go/src/github.com/decred/dcrd && env GO111MODULE=on go install . .\cmd\...

RUN dcrd --version
