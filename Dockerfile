FROM golang:1.26 AS builder

WORKDIR /src

ARG VERSION=dev
ARG COMMIT=unknown

COPY . .

RUN CGO_ENABLED=0 go build \
	-ldflags "-X main.version=${VERSION} -X main.hash=${COMMIT}" \
	-o ./dist/solbot ./cmd

FROM alpine:latest

WORKDIR /srv/

COPY --from=builder /src/dist/solbot ./solbot

CMD [ "./solbot" ]
