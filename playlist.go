package main

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/url"
	"os"
	"path"

	"github.com/zmb3/spotify/v2"
)

type Song struct {
	Title   string   `json:"title"`
	Artists []string `json:"artists"`
	Image   string   `json:"image"`
}

func fromPlaylistItem(item spotify.PlaylistItem) *Song {
	artists := make([]string, len(item.Track.Track.Artists))
	for i, artist := range item.Track.Track.Artists {
		artists[i] = artist.Name
	}
	return &Song{
		Title:   item.Track.Track.Name,
		Artists: artists,
		Image:   item.Track.Track.Album.Images[0].URL,
	}
}

type Match struct {
	Winner *Song `json:"winner"`
	Loser  *Song `json:"loser"`
}

type Stage []Match

type Playlist struct {
	ID               string  `json:"id"`
	Url              string  `json:"url"`
	Songs            []*Song `json:"songs"`
	Stages           []Stage `json:"stages"`
	CurrentSelection []*Song `json:"currentSelection"`
	NextSelection    []*Song `json:"nextSelection"`
}

func (p *Playlist) nextPair() (*Song, *Song) {
	if len(p.CurrentSelection) >= 2 {
		p.NextSelection = append(p.NextSelection, p.CurrentSelection[0], p.CurrentSelection[1])
		p.CurrentSelection = p.CurrentSelection[2:]
		return p.NextSelection[len(p.NextSelection)-2], p.NextSelection[len(p.NextSelection)-1]
	}

	// append the last item to the next stage if the current stage has an odd number of items
	if len(p.CurrentSelection)%2 != 0 {
		p.NextSelection = append(p.NextSelection, p.CurrentSelection[len(p.CurrentSelection)-1])
	}
	p.CurrentSelection = p.NextSelection
	shuffle(p.CurrentSelection)
	p.NextSelection = make([]*Song, 0, len(p.CurrentSelection)/2)
	p.Stages = append(p.Stages, Stage{})

	if len(p.CurrentSelection) == 1 {
		return p.CurrentSelection[0], nil
	}
	return p.nextPair()
}

func (p *Playlist) selected(selection int) {
	winner, loser := 1, 2
	if selection == 1 {
		winner, loser = 2, 1
	}
	p.Stages[len(p.Stages)-1] = append(p.Stages[len(p.Stages)-1], Match{
		Winner: p.NextSelection[len(p.NextSelection)-winner],
		Loser:  p.NextSelection[len(p.NextSelection)-loser],
	})

	if selection == 2 {
		p.NextSelection[len(p.NextSelection)-2] = p.NextSelection[len(p.NextSelection)-1]
	}
	p.NextSelection = p.NextSelection[:len(p.NextSelection)-1]
}

func (p *Playlist) save() error {
	playlistJson, err := json.MarshalIndent(playlist, "", "\t")
	if err != nil {
		return err
	}

	if err := os.WriteFile("game.json", playlistJson, 0o644); err != nil {
		return err
	}
	return nil
}

var playlist *Playlist

func loadPlaylist(playlist_url string) (*Playlist, error) {
	playlist := &Playlist{Url: playlist_url, Stages: []Stage{{}}}

	var err error
	playlist.ID, err = getPlaylistIdFromURL(playlist_url)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tracks, err := spClient.GetPlaylistItems(ctx, spotify.ID(playlist.ID))
	if err != nil {
		return nil, err
	}

	for page := 1; ; page++ {
		songs := make([]*Song, len(tracks.Items))
		for i, item := range tracks.Items {
			songs[i] = fromPlaylistItem(item)
		}

		playlist.Songs = append(playlist.Songs, songs...)

		err := spClient.NextPage(context.Background(), tracks)
		if errors.Is(err, spotify.ErrNoMorePages) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	shuffle(playlist.Songs)
	playlist.CurrentSelection = make([]*Song, len(playlist.Songs))
	copy(playlist.CurrentSelection, playlist.Songs)
	playlist.NextSelection = make([]*Song, 0, len(playlist.Songs)/2)
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
