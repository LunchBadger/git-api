# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang:1.9.4-alpine3.7

RUN mkdir /app
RUN mkdir -p /go/src/github.com/LunchBadger/git-api
ADD . /go/src/github.com/LunchBadger/git-api
WORKDIR /go/src/github.com/LunchBadger/git-api

RUN go build -o /app/main .

CMD ["/app/main"] 

# Document that the service listens on port 8080.
EXPOSE 8080