FROM golang:1.20

WORKDIR /src

RUN curl -fsSL https://deb.nodesource.com/setup_19.x | bash -
RUN apt-get update && apt-get install -y python nodejs build-essential 

COPY . .

RUN cd web && npm install && npm run staging
RUN go build -o ./dist/admin ./cmd/
