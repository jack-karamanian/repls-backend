FROM golang

RUN go get github.com/gorilla/websocket
RUN go get github.com/docker/docker/client

ADD . /app

WORKDIR /app
RUN go build main.go server.go container.go
EXPOSE 8080
CMD ./main