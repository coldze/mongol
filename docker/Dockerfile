FROM golang:alpine

RUN apk update && apk add bash git && go get -u github.com/kardianos/govendor

RUN go get github.com/coldze/mongol && \
    cd $GOPATH/src/github.com/coldze/mongol && \
    govendor sync && \
    go install && \
    rm -rf $GOPATH/src/github.com
