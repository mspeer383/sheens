FROM golang:1.24.1-alpine3.21

RUN mkdir -p $GOPATH/src/github.com/sheens

COPY . $GOPATH/src/github.com/sheens

WORKDIR $GOPATH/src/github.com/sheens

RUN go get ./... && make prereqs
