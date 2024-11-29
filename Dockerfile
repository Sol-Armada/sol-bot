FROM golang:1.22 AS builder

WORKDIR /src

COPY . .

RUN go build -o ./dist/solbot ./cmd

FROM golang:latest

WORKDIR /srv/

COPY --from=builder /src/dist/solbot ./solbot

CMD [ "./solbot" ]
