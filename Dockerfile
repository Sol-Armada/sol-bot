FROM golang:1.20 AS builder

ARG VITE_API_BASE_URL
ARG VITE_DISCORD_AUTH_URL

WORKDIR /src

RUN curl -fsSL https://deb.nodesource.com/setup_19.x | bash -
RUN apt-get update && apt-get install -y python nodejs build-essential 
RUN npm install --global yarn
RUN mkdir -p ./build
RUN mkdir -p ./web/dist
RUN mkdir -p ./dist

COPY . .

RUN git clone https://github.com/sol-armada/sol-bot-web.git ./build/web
RUN echo "${VITE_API_BASE_URL}\n" >> ./build/web/.env.production
RUN echo "${VITE_DISCORD_AUTH_URL}" >> ./build/web/.env.production
RUN cd ./build/web/ && yarn install
RUN cd ./build/web/ && yarn build
RUN cp -R ./build/web/dist/* ./web/dist/

RUN go build -o ./dist/admin ./cmd/

FROM golang:latest

WORKDIR /srv/

COPY --from=builder /src/dist/admin ./admin

CMD [ "./admin" ]
