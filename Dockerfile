FROM golang:1.11

WORKDIR /go/src/github.com/jfixby/dcrregtest
COPY . .

RUN apt-get update && apt-get upgrade -y && apt-get install -y rsync

RUN git clone https://github.com/decred/dcrd /go/src/github.com/decred/dcrd
RUN git checkout release-v1.4

RUN cd /go/src/github.com/decred/dcrd
RUN dir
RUN env GO111MODULE=on go install . .\cmd\...
RUN dir
RUN cd /go/src/github.com/jfixby/dcrregtest
RUN dir

RUN dcrd --version
