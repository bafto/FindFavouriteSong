FROM golang:alpine as build

RUN apk add --no-cache --update gcc g++

COPY . /app
WORKDIR /app
ENV CGO_ENABLED=1
RUN go build -ldflags "-s -w" -o FindFavouriteSong .

FROM alpine as run

RUN apk add --no-cache tzdata

ENV TZ=Europe/Berlin

COPY --from=build /app/FindFavouriteSong /app/FindFavouriteSong
COPY select_songs.html select_playlist.html stats.html winner.html /app/

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

CMD [ "/entrypoint.sh" ]
