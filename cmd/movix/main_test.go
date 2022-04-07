package main

import (
	"os"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestMakeConfig(t *testing.T) {
    t.Run("Make a config and read it back", func(t *testing.T) {

        // want := 4
        //
        // // Call the function you want to test.
        // got := 2+2
        //
        // // Assert that you got your expected response
        // if got != want {
        //     t.Fail()
        // }
    })
}

func test_config() *Config {
	return &Config{
		Mediapath: "",
		Perm: 644,
		Dbpath: "",
		Treshhold: 0, // this way we add all files
		LuaPluginPath: "",
		Move: false,
	}
}
func test_db() *gorm.DB {
	  db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	  if err != nil {
		  return nil
	  }
	  return db
}

func verifyentries(db *gorm.DB, t*testing.T) {
	paths := []string{
		"testdir/Bowling.for.Columbine.2002.720p.WEB-DL.H264-WEBiOS.mkv",
		"testdir/MasterChef.New.Zealand.S03E25.PDTV.x264-FiHTV.mp4",
	}
	for _, path := range(paths) {
		var entry Entry
		err := db. Where("path = ?", path).
		First(&entry).
		Error
		if err != nil {
			t.Fail()
		}
		if entry.Path != path {
			t.Fail()
		}
	}
}

func TestWalk(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		conf := test_config()
		db := test_db()
		walker(db, conf, "testdir")
		verifyentries(db, t)
	})
}

func TestMove(t *testing.T){
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		conf := test_config()
		conf.Move = true
		conf.Mediapath = "testdir2"
		defer os.RemoveAll("testdir2")
		db := test_db()
		walker(db, conf, "testdir")
		verifyentries(db, t)
	})
}

// can we make this fuzzy?
func test_entry() *Entry {
	return &Entry{
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

func get_entry(db *gorm.DB, path string, t *testing.T) *Entry {
	var entry Entry 
	err := db.First(&entry, "path = ?", path).Error
	if err != nil {
		t.Fail()
	}
	return &entry
}

func TestWatched(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		conf := test_config()
		db := test_db()
		entry := test_entry()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(entry)
		updateWatched(db, conf, entry.Path, 0, true)
		updated := get_entry(db, entry.Path, t)
		if !updated.Watched {
			t.Fail()
		}
	})
}

func TestWatchedpartial(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		conf := test_config()
		db := test_db()
		entry := test_entry()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(entry)
		updateWatched(db, conf, entry.Path, 40, true)
		updated := get_entry(db, entry.Path, t)
		if updated.Watched {
			t.Fail()
		}
		if updated.Offset != 40 {
			t.Fail()
		}
	})
}

func test_episode() (*Episode, *Series) {
	entry := test_entry()
	series := test_series()
	episode := &Episode{
		Part: 3,
		Season: 2,
		SeriesId: series.Id,
		Series: *series,
		EntryID: entry.Id,
		Entry: *entry,
	}
	return episode, series
}

func test_series() *Series{
	return &Series{
		Id: 22,
		Name: "Alias",
		LastWatched: time.Now(),
		Prio: 3,
		Subscribed: true,
	}
}

func TestNextTv(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		episode, series := test_episode()
		db := test_db()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
	})
}

func test_movie() *Movie{
	entry := test_entry()
	return &Movie{
		EntryID: entry.Id,
		Entry: *entry,
	}
}

func TestNextMovies(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		episode, series := test_episode()
		db := test_db()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)

		movie := test_movie()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(movie)
	})
}
func TestNext(t *testing.T) {
		movie := test_movie()
		db := test_db()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(movie)
}

func TestSkip(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
		episode, series := test_episode()
		db := test_db()
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
		db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
	// we need to insert series, not entries
	})
}

func TestUrl(t *testing.T) {
	t.Run("Walk a directory and add files to db", func(t *testing.T) {
	})
}
