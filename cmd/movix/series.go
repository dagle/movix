package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"gorm.io/gorm"
)

type Episode struct {
	gorm.Model
	Part int64
	Season int64
	SeriesId int
	Series Series
	EntryID int
	Entry Entry
};

type Series struct {
	Id int
	Name string
	LastWatched time.Time // to keep the episode in oder
	Prio int // maybe in the future, 
	Subscribed bool
	// LastEpisode int
	// more stuff to come
}

func get_series(title string) (*Series, error) {
	tmdbClient, err := tmdb.Init(APIKEY)
	if err != nil {
		return nil, err
	}
	search, err := tmdbClient.GetSearchTVShow(title, nil)
	if err != nil {
		return nil, err
	}
	// compare
	lowname := strings.ToLower(title)
	id := search.Results[0].ID
	// XXX this feels like a hack
	for _, r := range search.Results {
		if strings.ToLower(r.Name) == lowname || strings.ToLower(r.OriginalName) == lowname {
			id = r.ID
			break;
		}
	}
	show_details, err := tmdbClient.GetTVDetails(int(id), nil)
	if err != nil {
		return nil, err
	}
	series := &Series {
		Id: int(id),
		Name: show_details.Name,
		Subscribed: true,
		// LastEpisode: show_details.LastEpisodeToAir.EpisodeNumber,
	}
	return series, nil
}

func (series *Series) get_episode(path string, season, episodenum int64) (*Episode, error) {
	tmdbClient, err := tmdb.Init(APIKEY)
	if err != nil {
		return nil, err
	}

	show_details, err := tmdbClient.GetTVDetails(series.Id, nil)
	if err != nil {
		return nil, err
	}
	episode_details, err := tmdbClient.GetTVEpisodeDetails(series.Id, int(season), int(episodenum), nil)
	if err != nil {
		return nil, err
	}

	length := get_filelength(path)
	if episodeAlmostEqual(length / 60, show_details.EpisodeRunTime, 0.85) {
		episode := &Episode{
			Entry:Entry{
				Id:      episode_details.ID, // we shouldn't really do this
				Path:    path,
				Length:  length,
				Name:    show_details.Name,
				Added:   time.Now(),
				Deleted: false,
				Offset:  0,
				Watched: false,
			},
			// Add another id here
			Part:    episodenum,
			Season:  season,
			SeriesId: series.Id,
			Series: *series,
		}
		return episode, nil
	}
	return nil, errors.New("file length doesn't match tmdb file length")
}

func (episode *Episode) Make_name(codec string) string {
	return fmt.Sprintf("%s S%dE%d %s.%s", episode.Series.Name, 
		episode.Season, episode.Part, episode.Entry.Name, codec)
}

func (episode *Episode) Move(conf *Config, codec string) error {
	filename := episode.Make_name(codec)
	ep := strconv.FormatInt(episode.Season, 10)
	dir := conf.Mediapath + "/tv/" + episode.Series.Name + "/" + ep
	os.MkdirAll(dir, conf.Perm)
	new_path := dir + filename
	log.Printf("Moving file %s to %s\n", episode.Entry.Path, new_path)
	err := os.Rename(episode.Entry.Path, new_path)
	if err != nil {
		return err
	}
	episode.Entry.Path = new_path
	return nil
}
