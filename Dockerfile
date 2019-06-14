FROM golang:1.11

WORKDIR /go/src/github.com/jfixby/dcrregtest
COPY . .

RUN apt-get update && apt-get upgrade -y && apt-get install -y rsync
RUN dir
RUN dir ../
RUN dir ../../
RUN env GO111MODULE=on go find ./...
RUN dir
RUN dir ../
RUN dir ../../

CMD [ "dcrregtest" ]
