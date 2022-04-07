package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	tmdb "github.com/cyruzin/golang-tmdb"
	"gorm.io/gorm"
)

type Movie struct {
	gorm.Model
	EntryID int64
	Entry Entry
}

func get_movie(path, title string, treshhold float64) (*Movie, error) {
	tmdbClient, err := tmdb.Init(APIKEY)
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
	length := get_filelength(path)
	if almostEqual(length / 60, float64(movie_details.Runtime), treshhold) {
		movie := &Movie {
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
		}
		return movie, nil
	}
	return nil, errors.New("file length doesn't match tmdb file length")
}

// maybe escape filenames etc
func (movie *Movie) Make_name(codec string) string {
	return fmt.Sprintf("%s.%s", movie.Entry.Name, codec)
}

func (movie *Movie) Move(conf *Config, codec string) error {
	filename := movie.Make_name(codec)
	dir := conf.Mediapath + "/movies/"
	os.MkdirAll(dir, conf.Perm)
	new_path := dir + filename
	log.Printf("Moving file %s to %s\n", movie.Entry.Path, new_path)
	err := os.Rename(movie.Entry.Path, new_path)
	if err != nil {
		return err
	}
	movie.Entry.Path = new_path
	return nil
}
