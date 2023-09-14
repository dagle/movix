package movix

import (
	"errors"
	"fmt"
	"time"

	"database/sql"

	tmdb "github.com/cyruzin/golang-tmdb"
)

type Movie struct {
	Id      int64
	EntryID string
	Entry   Entry
}

type Movies struct{}

func (m *Movies) InitDb(db *sql.DB) error {
	sqlStmt :=
	`create table movie (
      id int not null primary key,
      entryid int not null,
      FOREIGN KEY (entryid) REFERENCES entry(id))
    `
	_, err := db.Exec(sqlStmt)
	return err
}

const MOVIE_KEY string = "e130a499c97798cfac3ffb5d0e2cc1be"

func make_movie_name(movie *tmdb.MovieDetails) string {
	return fmt.Sprintf("%s (%s)", movie.Title, movie.ReleaseDate)
}

// This is so we can identify a movie.
func make_movie_id(movie *tmdb.MovieDetails) string {
	return fmt.Sprintf("movie: %d", movie.ID)
}

func get_movie(path, title string, runtime *Runtime) (*Movie, error) {
	tmdbClient, err := tmdb.Init(MOVIE_KEY)
	if err != nil {
		return nil, err
	}
	search, err := tmdbClient.GetSearchMovies(title, nil)
	if err != nil {
		return nil, err
	}
	id := search.Results[0].ID
	movie_details, err := tmdbClient.GetMovieDetails(int(id), nil)
	if err != nil {
		return nil, err
	}

	length := 0.0
	if runtime.VerifyLength {
		length = get_filelength(path)
		if !almostEqual(length/60, float64(movie_details.Runtime), runtime.Treshhold) {
			return nil, errors.New("file length doesn't match tmdb file length")
		}
	}
	now := time.Now().Unix()
	name := make_movie_name(movie_details)
	eid := make_movie_id(movie_details)
	return &Movie{
		Entry: Entry{
			Id:           eid,
			Path:         path,
			Length:       length,
			Name:         name,
			Added:        now,
			Deleted:      false,
			Offset:       0,
			Watched:      false,
			Watched_date: now, // this date is bogus but only so we can compare
		},
		EntryID: eid,
	}, nil
}

// TODO: Add a overwrite flag
func (m *Movies) Add(db *sql.DB, runtime *Runtime, path string, info *FileInfo) error {
	// this shouldn't be a part of add
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

	movie, err := get_movie(path, info.Title, runtime)
	if err != nil {
		return err
	}

	entry, err = GetEntry(db, movie.EntryID)
	if err == nil && entry != nil {
		if entry.Deleted {
			entry.Deleted = false
			entry.Added = time.Now().Unix()
			entry.Path = movie.Entry.Path
			_, err = entry.Save(db)
			return err
		} else {
			return nil
		}
	}

	movie_stmt, err := db.Prepare("insert into movie(id, entryid) values(?, ?)");
	if err != nil {
		return err
	}
	defer movie_stmt.Close()

	movie_stmt.Exec(movie.Id, movie.EntryID)
	return nil
}
func Make_movies() *Movies {
	return &Movies{}
}

func (m *Movies) FsMatch(info *FileInfo) bool {
	filetypes := []string{"mkv", "avi", "mp4", "mov", "wmv", "peg"}
	if !matchOne(info.Container, filetypes) {
		return false
	}
	if info.Mimetype[:5] == "video" && info.Type == "movie" {
		return true
	}
	return false
}

func (m *Movies) Next(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`select entry.name from movie 
		join entry
			ON movie.entryid = entry.id
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

func (m *Movies) Select(db *sql.DB, name string) (*Entry, error) {
	stmt, err := db.Prepare(`select entry.id, 
		entry.length,
		entry.path,
		entry.name,
		entry.added,
		entry.offset,
		entry.deleted,
		entry.watched,
		entry.watched_date
	from movie 
		join entry
			ON movie.entryid = entry.id
		where entry.watched = 0 and entry.deleted = 0 and entry.name = ?")
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
