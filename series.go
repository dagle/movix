package movix

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
	"gorm.io/gorm/clause"
)

type Episode struct {
	gorm.Model
	Part int64
	Season int64
	SeriesId int
	Series Series
	EntryID int64
	Entry Entry
};

type Series struct {
	Id int
	Name string
	LastWatched time.Time // to keep the episode in oder
	Prio int // maybe in the future, 
	Subscribed bool
}

type Tv struct{}

const SERIES_KEY string = "e130a499c97798cfac3ffb5d0e2cc1be"

func get_series(title string) (*Series, error) {
	tmdbClient, err := tmdb.Init(SERIES_KEY)
	if err != nil {
		return nil, err
	}
	search, err := tmdbClient.GetSearchTVShow(title, nil)
	if err != nil {
		return nil, err
	}
	// compare
	lowname := strings.ToLower(title)

	// XXX this feels like a hack
	id := search.Results[0].ID
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

func episodeAlmostEqual(a float64, bs []int, threshold float64) bool {
	for _, b := range bs {
		if almostEqual(a, float64(b), threshold) {
			return true
		}
	}
	return false
}

func (series *Series) get_episode(path string, season, episodenum int64, treshhold float64, verify_length bool) (*Episode, error) {
	tmdbClient, err := tmdb.Init(SERIES_KEY)
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

	length := 0.0
	if verify_length { 
		length = get_filelength(path)
		if !episodeAlmostEqual(length / 60, show_details.EpisodeRunTime, treshhold) {
			return nil, errors.New("file length doesn't match tmdb file length")
		}
	} 
	return &Episode{
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
	}, nil
}

func Make_series() *Tv {
	return &Tv{}
}

func (tv *Tv) FsMatch(info *FileInfo) bool {
	filetypes := []string{"mkv", "avi", "mp4", "mov", "wmv", "peg"}
	if !matchOne(info.Container, filetypes) {
		return false
	}
	if info.Mimetype[:5] == "video" && info.Type == "episode" {
		return true
	}
	return false
}

func (tv *Tv) Add(db *gorm.DB, runtime *Runtime, path string, info *FileInfo) error {
	series, err := get_series(info.Title)
	if err != nil {
		return err
	}
	episode, err := series.get_episode(path, info.Season, info.Episode, runtime.MatchLength, runtime.VerifyLength)
	if err != nil {
		return err
	}
	// 	if conf.Move {
	// 		episode.Move(conf, guessed.Codec)
	// 	}
	db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
	db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
	return nil
}

func (tv *Tv) Next(db *gorm.DB) ([]string, error) {
	var episodes []Episode
	err := db.Joins("Series").Joins("Entry").
		Where("watched = ? and subscribed = ? and deleted = ?", false, true, false).
		Order("last_watched, season, part").
		Group("Series.id").
		Find(&episodes).
		Error

	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range episodes {
		names = append(names, e.Series.Name)
	}
	return names, nil
}

func (tv *Tv) Select(db *gorm.DB, name string) (*Entry, error){
	var episode Episode
	err := db.Joins("Series").Joins("Entry").
		Where("watched = ? and deleted = ? and Series.name = ?", false, false, name).
		Order("last_watched, season, part").
		Group("Series.id").
		First(&episode).
		Error
	if err != nil {
		return nil, err
	}

	return &episode.Entry, nil
}

// this should return an interator?
func (tv *Tv)skipUntil(db *gorm.DB, name string, season, episode int64) error {
	var episodes []Episode
	err := db.Joins("Series").Joins("Entry").
		Where("Series.name = ? and season < ? or (season = ? and part < ?)", name, season, season, episode).
		Find(&episodes).
		Error

	if err != nil {
		return err
	}
	for _, e := range episodes {
		e.Entry.Watched = true
		db.Save(&e.Entry)
	}
	return nil
}

func (tv *Tv)unmarkAfter(db *gorm.DB, name string, season, episode int) error {
	var episodes []Episode
	err := db.Joins("Series").Joins("Entry").
		Where("Series.name = ? and season > ? or (season = ? and part > ?)", name, season, season, episode).
		Find(&episodes).
		Error

	if err != nil {
		return err
	}
	for _, e := range episodes {
		e.Entry.Watched = false
		db.Save(&e.Entry)
	}
	return nil
}

func (episode *Episode) make_name(codec string) string {
	return fmt.Sprintf("%s S%dE%d %s.%s", episode.Series.Name, 
		episode.Season, episode.Part, episode.Entry.Name, codec)
}

func (episode *Episode) Move(runtime *Runtime, codec string) error {
	filename := episode.make_name(codec)
	ep := strconv.FormatInt(episode.Season, 10)
	dir := runtime.Mediapath + "/tv/" + episode.Series.Name + "/" + ep
	os.MkdirAll(dir, runtime.Perm)
	new_path := dir + filename
	log.Printf("Moving file %s to %s\n", episode.Entry.Path, new_path)
	err := os.Rename(episode.Entry.Path, new_path)
	if err != nil {
		return err
	}
	episode.Entry.Path = new_path
	return nil
}
