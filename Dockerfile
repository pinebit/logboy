FROM golang:1.20-alpine as builder

WORKDIR /service

COPY go.mod ./
COPY go.sum ./
COPY *.go ./
COPY app/ ./app/

RUN go mod download
RUN go build -o /lognite

FROM alpine

COPY --from=builder /lognite /lognite

CMD [ "/lognite" ]
