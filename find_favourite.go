package main

import (
	"context"
	"errors"
	"math/rand"
	"net/url"
	"path"

	"github.com/zmb3/spotify/v2"
)

type Playlist struct {
	ID           string
	Url          string
	Items        []spotify.PlaylistItem
	stage        int
	i            int
	currentStage []spotify.PlaylistItem
	nextStage    []spotify.PlaylistItem
}

func (p *Playlist) nextPair() (*spotify.PlaylistItem, *spotify.PlaylistItem) {
	if len(p.currentStage) == 1 {
		return &p.currentStage[0], nil
	}

	if p.i <= len(p.currentStage)-2 {
		p.i += 2
		return &p.currentStage[p.i-2], &p.currentStage[p.i-1]
	}
	// append the last item to the next stage if the current stage has an odd number of items
	if len(p.currentStage)%2 != 0 {
		p.nextStage = append(p.nextStage, p.currentStage[len(p.currentStage)-1])
	}
	p.currentStage = p.nextStage
	shuffle(p.currentStage)
	p.nextStage = make([]spotify.PlaylistItem, 0, len(p.currentStage)/2)
	p.stage++
	p.i = 0
	return p.nextPair()
}

func (p *Playlist) selected(selection int) {
	p.nextStage = append(p.nextStage, p.currentStage[p.i-2+selection])
}

var playlist *Playlist

func loadPlaylist(playlist_url string) (*Playlist, error) {
	playlist := &Playlist{Url: playlist_url}

	var err error
	playlist.ID, err = getPlaylistIdFromURL(playlist_url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tracks, err := spClient.GetPlaylistItems(ctx, "7lXG7V2pA7gCWviuDSIeFX")
	if err != nil {
		return nil, err
	}

	for page := 1; ; page++ {
		playlist.Items = append(playlist.Items, tracks.Items...)

		err := spClient.NextPage(context.Background(), tracks)
		if errors.Is(err, spotify.ErrNoMorePages) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	shuffle(playlist.Items)
	playlist.currentStage = make([]spotify.PlaylistItem, len(playlist.Items))
	copy(playlist.currentStage, playlist.Items)
	playlist.nextStage = make([]spotify.PlaylistItem, 0, len(playlist.Items)/2)
	return playlist, nil
}

func getPlaylistIdFromURL(u string) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}
	return path.Base(parsed.Path), nil
}

func shuffle[T any](items []T) {
	for i := range items {
		j := rand.Intn(i + 1)
		items[i], items[j] = items[j], items[i]
	}
}
