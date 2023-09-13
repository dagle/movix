package movix

import (
	"errors"
	"fmt"
	// "os"
	// "strconv"
	"strings"
	"time"

	"database/sql"
	tmdb "github.com/cyruzin/golang-tmdb"
)

type Episode struct {
	// gorm.Model
	Part     int64
	Season   int64
	SeriesId int // the id to the tmdb, maybe not needed
	Series   Series
	EntryID  int64
	Entry    Entry
}

type Series struct {
	Id           int
	Name         string
	Last_updated int64
	Last_watched int64
	Prio         int // maybe in the future,
	Subscribed   bool
}

func (t *Tv) InitDB(db *sql.DB) error {
	sqlStmt :=
		`create table episode (
      id integer not null primary key,
      part int,
      season int,
      seriesid int,
      entryid,
      FOREIGN KEY (entryid) REFERENCES entry(id))
    `
	_, err := db.Exec(sqlStmt)
	return err
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
			break
		}
	}
	show_details, err := tmdbClient.GetTVDetails(int(id), nil)
	if err != nil {
		return nil, err
	}
	series := &Series{
		Id:           int(id),
		Name:         show_details.Name,
		Subscribed:   true,
		Last_updated: time.Now().Unix(),
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

func make_name(show string, season int, episode int, name string) string {
	return fmt.Sprintf("%s S%dE%d %s", show,
		season, episode, name)
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
		if !episodeAlmostEqual(length/60, show_details.EpisodeRunTime, treshhold) {
			return nil, errors.New("file length doesn't match tmdb file length")
		}
	}
	now := time.Now().Unix()
	return &Episode{
		Entry: Entry{
			Id: episode_details.ID, // we shouldn't really do this
			// RemoteId: episode_details.ID,
			Path:         path,
			Length:       length,
			Name:         make_name(show_details.Name, int(season), int(episodenum), episode_details.Name),
			Added:        now,
			Deleted:      false,
			Offset:       0,
			Watched:      false,
			Watched_date: now, // this date is bogus but only so we can compare
		},
		// Add another id here
		Part:     episodenum,
		Season:   season,
		SeriesId: series.Id,
		Series:   *series,
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

func (tv *Tv) Add(db *sql.DB, runtime *Runtime, path string, info *FileInfo) error {
	// series, err := get_series(info.Title)
	// if err != nil {
	// 	return err
	// }
	// episode, err := series.get_episode(path, info.Season, info.Episode, runtime.MatchLength, runtime.VerifyLength)
	// if err != nil {
	// 	return err
	// }
	// 	if conf.Move {
	// 		episode.Move(conf, guessed.Codec)
	// 	}
	// db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
	// db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
	return nil
}

func (tv *Tv) Next(db *sql.DB) ([]string, error) {
	var episodes []Episode
	// err := db.Joins("Series").Joins("Entry").
	// 	Where("watched = ? and deleted = ?", false, false).
	// 	Order("last_watched, season, part").
	// 	// Distinct("Series.id").
	// 	Group("Series.id").
	// 	Find(&episodes).
	// 	Error
	//
	// if err != nil {
	// 	return nil, err
	// }
	//
	var names []string
	for _, e := range episodes {
		names = append(names, e.Series.Name)
	}
	return names, nil
}

func (tv *Tv) Select(db *sql.DB, name string) (*Entry, error) {
	var episode Episode
	// err := db.Joins("Series").Joins("Entry").
	// 	Where("watched = ? and deleted = ? and Series.name = ?", false, false, name).
	// 	Order("last_watched, season, part").
	// 	Group("Series.id").
	// 	First(&episode).
	// 	Error
	// if err != nil {
	// 	return nil, err
	// }
	// var series Series
	// e := db.Where("name = ?", name).First(&series).Error
	// if e != nil {
	// 	return nil, e
	// }
	//
	// series.Last_updated = time.Now().Unix()
	// db.Save(&series)

	return &episode.Entry, nil
}

// this should return an interator?
func (tv *Tv) skipUntil(db *sql.DB, name string, season, episode int64) error {
	// var episodes []Episode
	// err := db.Joins("Series").Joins("Entry").
	// 	Where("Series.name = ? and season < ? or (season = ? and part < ?)", name, season, season, episode).
	// 	Find(&episodes).
	// 	Error
	//
	// if err != nil {
	// 	return err
	// }
	// for _, e := range episodes {
	// 	e.Entry.Watched = true
	// 	db.Save(&e.Entry)
	// }
	return nil
}

func (tv *Tv) unmarkAfter(db *sql.DB, name string, season, episode int64) error {
	// var episodes []Episode
	// err := db.Joins("Series").Joins("Entry").
	// 	Where("Series.name = ? and season > ? or (season = ? and part > ?)", name, season, season, episode).
	// 	Find(&episodes).
	// 	Error
	//
	// if err != nil {
	// 	return err
	// }
	// for _, e := range episodes {
	// 	e.Entry.Watched = false
	// 	db.Save(&e.Entry)
	// }
	return nil
}

// func (episode *Episode) make_fsname(codec string) string {
// 	return fmt.Sprintf("%s S%dE%d %s.%s", episode.Series.Name,
// 		episode.Season, episode.Part, episode.Entry.Name, codec)
// }

// func (episode *Episode) Move(runtime *Runtime, codec string) error {
// 	filename := episode.make_fsname(codec)
// 	ep := strconv.FormatInt(episode.Season, 10)
// 	dir := runtime.Mediapath + "/tv/" + episode.Series.Name + "/" + ep
// 	os.MkdirAll(dir, runtime.Perm)
// 	new_path := dir + filename
// 	Log("Moving file %s to %s\n", episode.Entry.Path, new_path)
// 	err := os.Rename(episode.Entry.Path, new_path)
// 	if err != nil {
// 		return err
// 	}
// 	episode.Entry.Path = new_path
// 	return nil
// }
