FROM golang:1.11

WORKDIR /go/src/github.com/jfixby/dcrregtest
COPY . .

RUN apt-get update && apt-get upgrade -y && apt-get install -y rsync

RUN env GO111MODULE=on go install . ./cmd/...

EXPOSE 9108

CMD [ "dcrregtest" ]
