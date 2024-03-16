package main

import (
	"context"
	"errors"
	"log"
	"net/url"
	"path"
	"time"

	"github.com/bafto/FindFavouriteSong/auth"
	"github.com/zmb3/spotify/v2"
)

const timeout = time.Second * 7

func main() {
	client, err := auth.Authenticate()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tracks, err := client.GetPlaylistItems(ctx, "7lXG7V2pA7gCWviuDSIeFX")
	if err != nil {
		log.Fatal(err)
	}

	tctx := context.Background()
	log.Printf("Playlist has %d tracks\n", tracks.Total)
	for page := 1; ; page++ {
		log.Printf("Page %d has %d tracks\n", page, len(tracks.Items))
		err := client.NextPage(tctx, tracks)
		if errors.Is(err, spotify.ErrNoMorePages) {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
	}
}

func getPlaylistIdFromURL(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return path.Base(parsed.Path), nil
}
