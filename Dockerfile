FROM golang:1.21

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . ./
RUN go build -v -o /usr/local/bin/app ./...

RUN mkdir -p /usr/src/app/external

EXPOSE 8080

CMD ["app"]