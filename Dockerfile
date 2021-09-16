FROM golang:1.16-alpine

ENV GO111MODULE="on"
ENV GOROOT="/usr/local/go"

WORKDIR /app

COPY . .
RUN go mod download

RUN go build -o /cardinanny cmd/cardinanny/cardinanny.go
RUN go build -o /inject-cardinality cmd/cardinality-injector/inject-cardinality.go

EXPOSE 8080

# TODO Copy over bins to new stage

CMD [ "/cardinanny" ]