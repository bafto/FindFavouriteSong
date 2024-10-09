FROM golang:alpine as build

COPY . /app
WORKDIR /app
RUN go build -ldflags "-s -w" -o FindFavouriteSong .

FROM alpine as run

RUN apk add --no-cache tzdata

ENV TZ=Europe/Berlin

COPY --from=build /app/FindFavouriteSong /app/FindFavouriteSong
COPY select_songs.html select_playlist.html stats.html winner.html /app/

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

CMD [ "/entrypoint.sh" ]
