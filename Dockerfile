FROM golang:alpine as build

COPY . /app
WORKDIR /app
RUN go build -ldflags "-s -w" -o FindFavouriteSong .

FROM alpine as run

RUN apk add --no-cache tzdata

ENV TZ=Europe/Berlin

COPY --from=build /app/FindFavouriteSong /app/FindFavouriteSong

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

CMD [ "/entrypoint.sh" ]
