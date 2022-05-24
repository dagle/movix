package movix

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Movie struct {
	gorm.Model
	EntryID int64
	Entry Entry
}

type Movies struct{}

const MOVIE_KEY string = "e130a499c97798cfac3ffb5d0e2cc1be"

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
		if !almostEqual(length / 60, float64(movie_details.Runtime), runtime.Treshhold) {
			return nil, errors.New("file length doesn't match tmdb file length")
		}
	} 
	return &Movie {
		Entry:Entry {
			Id: movie_details.ID,
			Path:    path,
			Length:  length,
			Name:    movie_details.Title,
			Added:   time.Now(),
			Deleted: false,
			Offset:  0,
			Watched: false,
		},
		EntryID: movie_details.ID,
	}, nil
}

func (m *Movies) Add(db *gorm.DB, runtime *Runtime, path string, info *FileInfo) error {
	movie, err := get_movie(path, info.Title, runtime)
	if err != nil {
		return err
	}
	// if conf.Move {
	// 	movie.Move(conf, guessed.Codec)
	// }

	db.Clauses(clause.OnConflict{DoNothing: true}).Create(movie)
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

func (m *Movies) Next(db *gorm.DB) ([]string, error){
	var movies []Movie
	err := db.Joins("Entry").
		Where("watched = ? and deleted = ?", false, false).
		Find(&movies).
		Error
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range movies {
		names = append(names, e.Entry.Name)
	}
	return names, nil
}

func (m *Movies) Select(db *gorm.DB, name string) (*Entry, error){
	var entry Entry
	err := db. Where("watched = ? and deleted = ? and name = ?", false, false, name).
		First(&entry).
		Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// maybe escape filenames etc
func (movie *Movie) make_name(codec string) string {
	return fmt.Sprintf("%s.%s", movie.Entry.Name, codec)
}

func (movie *Movie) move(runtime *Runtime, codec string) error {
	filename := movie.make_name(codec)
	dir := runtime.Mediapath + "/movies/"
	os.MkdirAll(dir, runtime.Perm)
	new_path := dir + filename
	log.Printf("Moving file %s to %s\n", movie.Entry.Path, new_path)
	err := os.Rename(movie.Entry.Path, new_path)
	if err != nil {
		return err
	}
	movie.Entry.Path = new_path
	return nil
}
