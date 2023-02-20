FROM golang:1.19-alpine

WORKDIR /service

COPY * ./

RUN apk add build-base
RUN go mod download
RUN go build -o /obry

EXPOSE 8080

CMD [ "/obry" ]
