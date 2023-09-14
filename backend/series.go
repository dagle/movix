package movix

import (
	"fmt"
	"strings"
	"time"

	"database/sql"
	tmdb "github.com/cyruzin/golang-tmdb"
)

type Episode struct {
	Id       int64
	Part     int64
	Season   int64
	SeriesId int // the id to the tmdb, maybe not needed
	Series   Series
	EntryID  string
	Entry    Entry
}

type Series struct {
	Id           int
	Name         string
	Last_updated int64
	Last_watched int64
}

func (t *Tv) InitDB(db *sql.DB) error {
	sqlSeriesStmt :=
		`create table series (
      id integer not null primary key,
      name text not null,
      last_updated int,
      last_watched int
      )`

	_, err := db.Exec(sqlSeriesStmt)

	if err != nil {
		return err
	}

	sqlEpisodeStmt :=
		`create table episode (
      id integer not null primary key,
      entryid int not null,
      part int not null,
      season int not null,
      seriesid int not null,
      FOREIGN KEY (entryid) REFERENCES entry(id),
      FOREIGN KEY (seriesid) REFERENCES series(id)
  )`
	_, err = db.Exec(sqlEpisodeStmt)

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
		Id:   int(id),
		Name: show_details.Name,
		// Subscribed:   true,
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

func make_episode_name(show string, season int, episode int, name string) string {
	return fmt.Sprintf("%s S%dE%d %s", show,
		season, episode, name)
}

func make_episode_id(episode *tmdb.TVEpisodeDetails) string {
	return fmt.Sprintf("episode: %d", episode.ID)
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

	length := get_filelength(path)

	// TODO: Fix this, this episodes should have a length now
	// if verify_length {
	// 	if !episodeAlmostEqual(length/60, show_details.EpisodeRunTime, treshhold) {
	// 		return nil, errors.New("file length doesn't match tmdb file length")
	// 	}
	// }

	eid := make_episode_id(episode_details)
	now := time.Now().Unix()
	return &Episode{
		Entry: Entry{
			Id:           eid,
			Path:         path,
			Length:       length,
			Name:         make_episode_name(show_details.Name, int(season), int(episodenum), episode_details.Name),
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
		EntryID: eid,
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
	entry, err := GetEntryPath(db, path)

	// if we have the file in the db,
	if err == nil && entry != nil {
		if entry.Deleted {
			entry.Deleted = false
			entry.Added = time.Now().Unix()
			_, err = entry.Save(db)
			return err
		} else {
			return nil
		}
	}

	series, err := get_series(info.Title)
	if err != nil {
		fmt.Println(err)
		return err
	}
	episode, err := series.get_episode(path, info.Season, info.Episode, runtime.MatchLength, runtime.VerifyLength)
	if err != nil {
		fmt.Println(err)
		return err
	}
	
	entry, err = GetEntry(db, episode.EntryID)
	if err == nil && entry != nil {
		if entry.Deleted {
			entry.Deleted = false
			entry.Added = time.Now().Unix()
			entry.Path = episode.Entry.Path
			_, err = entry.Save(db)
			return err
		} else {
			return nil
		}
	}

	series_stmt, err := db.Prepare("insert or replace into series(id, name, last_updated) values(?, ?, ?)")
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer series_stmt.Close()
	_, err = series_stmt.Exec(series.Id, series.Name, series.Last_updated)

	if err != nil {
		fmt.Println(err)
		return err
	}

	episode.Entry.Save(db)

	episode_stmt, err := db.Prepare("insert into episode(id, part, season, seriesid, entryid) values(?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer episode_stmt.Close()

	_, err = episode_stmt.Exec(episode.Id, episode.Part, episode.Season, episode.SeriesId, episode.EntryID)

	return err
}

func (tv *Tv) Next(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`select series.name from series 
		join episode
			ON episode.seriesid = series.id	
		join entry
			ON episode.entryid = entry.id
		where entry.watched = 0 and entry.deleted = 0
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}

	return names, nil
}

func (tv *Tv) Select(db *sql.DB, name string) (*Entry, error) {
	stmt, err := db.Prepare(`select entry.id, 
		entry.length,
		entry.path,
		entry.name,
		entry.added,
		entry.offset,
		entry.deleted,
		entry.watched,
		entry.watched_date
	from series 
		join episode
			ON episode.seriesid = series.id	
		join entry
			ON episode.entryid = entry.id
		where entry.watched = 0 and entry.deleted = 0  and series.name = ?
	`)

	if err != nil {
		return nil, err
	}

	defer stmt.Close()
	var entry Entry
	err = stmt.QueryRow(name).Scan(&entry.Id, &entry.Length, &entry.Path, &entry.Name,
		&entry.Added, &entry.Offset, &entry.Deleted, &entry.Watched, &entry.Watched_date)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (tv *Tv) skipUntil(db *sql.DB, name string, season, episode int64) error {
	stmt, err := db.Prepare(`update entry set
		entry.watched = true
	from entry 
		join episode
			ON entry.id = episode.entryid
		join series
			ON episode.seriesid = series.id
		where series.name = ? and (series.season < ? or (series.season = ? and series.part < ?))")
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, season, season, episode)

	return err
}

func (tv *Tv) unmarkAfter(db *sql.DB, name string, season, episode int64) error {
	stmt, err := db.Prepare(`update entry set
		entry.watched = true
	from entry 
		join episode
			ON entry.id = episode.entryid
		join series
			ON episode.seriesid = series.id
		where series.name = ? and (series.season > ? or (series.season = ? and series.part > ?))")
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, season, season, episode)

	return err
}
