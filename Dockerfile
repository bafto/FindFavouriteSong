FROM golang:alpine as build

COPY . /app
WORKDIR /app
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
RUN go generate
RUN go build -ldflags "-s -w" -o FindFavouriteSong .

FROM alpine as run

COPY --from=build /app/FindFavouriteSong .

CMD [ "./FindFavouriteSong" ]
