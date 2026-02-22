FROM golang:1.26 AS builder

WORKDIR /src

COPY . .

RUN go build -o ./dist/solbot ./cmd

FROM alpine:latest

WORKDIR /srv/

COPY --from=builder /src/dist/solbot ./solbot

CMD [ "./solbot" ]
