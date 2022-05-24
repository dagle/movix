package main

import (
	"os"
	"time"

	"testing"

	"github.com/dagle/movix"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestMakeConfig(t *testing.T) {
    t.Run("Make a config and read it back", func(t *testing.T) {
		dir := t.TempDir()
		conf := Conf(dir, "")
		make_config(dir, conf)
		conf2 := Conf(dir, "")
		if conf == conf2 {
			t.Fail()
		}
    })
}

func test_config() *movix.Runtime {
	return &movix.Runtime{
		Mediapath: "",
		Perm: 644,
		Dbpath: "",
		Treshhold: 0.9,
		Move: false,
		// MatchLength float64
		VerifyLength: false,
	}
}
func test_db(t *testing.T) *gorm.DB {
	// db, mock, err := sqlmock.New()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		return nil
	}
	db.AutoMigrate(&movix.Episode{})
	db.AutoMigrate(&movix.Series{})
	db.AutoMigrate(&movix.Movie{})
	t.Cleanup(func() {
		db.Migrator().DropTable(&movix.Episode{})
		db.Migrator().DropTable(&movix.Series{})
		db.Migrator().DropTable(&movix.Movie{})
	})	
	return db
}

var paths []string = []string{
	"Bowling.for.Columbine.2002.720p.WEB-DL.H264-WEBiOS.mkv",
	"Alias S01E01 Truth Be Told.avi",
}

func test_dir(t *testing.T) string {
	path := t.TempDir()
	for _, file := range(paths) {
		os.Create(path + "/" + file)
	}

	return path
}

func verifyentries(db *gorm.DB, t*testing.T, dir string) {
	for _, path := range(paths) {
		uri := dir + "/" + path
		var entry movix.Entry
		err := db.Where("path = ?", uri).
		First(&entry).
		Error
		if err != nil {
			t.Fail()
		}
		if entry.Path != uri {
			t.Fail()
		}
	}
}

func TestWalk(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		runtime := test_config()
		db := test_db(t)
		dir := test_dir(t)
		movies := movix.Make_movies()
		tv := movix.Make_series()
		err := movix.RunWalkers(db, runtime, dir, movies, tv)
		if err != nil {
			t.Fail()
		}
		verifyentries(db, t, dir)
	})
}

func TestMove(t *testing.T){
	t.Run("Add and move the files", func(t *testing.T) {
		runtime := test_config()
		db := test_db(t)
		dir := test_dir(t)
		target := t.TempDir()
		runtime.Move = true
		runtime.Mediapath = target
		movies := movix.Make_movies()
		tv := movix.Make_series()
		err := movix.RunWalkers(db, runtime, dir, movies, tv)
		if err != nil {
			t.Fail()
		}
		verifyentries(db, t, dir)
		// verifyentries(db, t, target)
	})
}

// can we make this fuzzy?
func test_entry() *movix.Entry {
	return &movix.Entry{
		Id: 7,
		Length: 98,
		Path: "testpath.mkv",
		Added: time.Now(),
		Offset: 0,
		Deleted: false,
		Watched: false,
		Watched_date: time.Now(),
	}
}

func get_entry(db *gorm.DB, path string, t *testing.T) *movix.Entry {
	var entry movix.Entry
	err := db.First(&entry, "path = ?", path).Error
	if err != nil {
		t.Fail()
	}
	return &entry
}

func TestWatched(t *testing.T) {
	t.Run("Watch an entry", func(t *testing.T) {
		conf := test_config()
		db := test_db(t)
		entry := test_entry()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(entry)
		movix.UpdateWatched(db, conf, entry.Path, 0, true)
		updated := get_entry(db, entry.Path, t)
		if !updated.Watched {
			t.Errorf("Failed to set entry to watched")
		}
	})
}
//
func TestWatchedpartial(t *testing.T) {
	t.Run("Partialy watch an entry", func(t *testing.T) {
		conf := test_config()
		db := test_db(t)
		entry := test_entry()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(entry)
		movix.UpdateWatched(db, conf, entry.Path, 40, false)
		updated := get_entry(db, entry.Path, t)
		if updated.Watched {
			t.Errorf("Entry shouldn't be considered watched")
		}
		if updated.Offset != 40 {
			t.Errorf("Entry offset not set to correct value")
		}
	})
}

func test_episode() (*movix.Episode, *movix.Series) {
	entry := test_entry()
	series := test_series()
	episode := &movix.Episode{
		Part: 3,
		Season: 2,
		SeriesId: series.Id,
		Series: *series,
		EntryID: entry.Id,
		Entry: *entry,
	}
	return episode, series
}

func test_series() *movix.Series{
	return &movix.Series{
		Id: 22,
		Name: "Alias",
		LastWatched: time.Now(),
		Prio: 3,
		Subscribed: true,
	}
}

func compareTv(ep *movix.Episode, entry *movix.Entry) bool {
	return ep.Entry == *entry
}

func TestNextTv(t *testing.T) {
	t.Run("Get the tv episode", func(t *testing.T) {
		episode, series := test_episode()
		db := test_db(t)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
		entry, err := movix.GetNext(db, series.Name, movix.Make_series(), movix.Make_movies())
		
		if err != nil {
			t.Fail()
		}

		if !compareTv(episode, entry) {
			t.Fail()
		}
	})
}

func test_movie() *movix.Movie{
	entry := test_entry()
	return &movix.Movie{
		EntryID: entry.Id,
		Entry: *entry,
	}
}

func compareMovie(movie *movix.Movie, entry *movix.Entry) bool {
	return movie.Entry == *entry
}

func TestNextMovies(t *testing.T) {
	t.Run("Get the next movie", func(t *testing.T) {
		db := test_db(t)
		movie := test_movie()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(movie)

		entry, err := movix.GetNext(db, movie.Entry.Name, movix.Make_series(), movix.Make_movies())
		
		if err != nil {
			t.Fail()
		}

		if !compareMovie(movie, entry) {
			t.Fail()
		}
	})
}

func TestNexts(t *testing.T) {
		db := test_db(t)
		movie := test_movie()
		episode, series := test_episode()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(movie)

		// verify the output
		GetNexts(db, movix.Make_movies(), movix.Make_series())
}

func TestSkip(t *testing.T) {
	t.Run("Skip a single episode using SkipUntil", func(t *testing.T) {
		episode, series := test_episode()
		db := test_db(t)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
		err := movix.SkipUntil(db, series.Name, episode.Season, episode.Part)
		if err != nil {
			t.Fail()
		}
		entry := get_entry(db, episode.Entry.Path, t)
		
		if !entry.Watched {
			t.Fail()
		}
	})
}
