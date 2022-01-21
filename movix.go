package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	// "io/fs"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	tmdb "github.com/cyruzin/golang-tmdb"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	// "gorm.io/gorm/logger"
)

const APIKEY = "e130a499c97798cfac3ffb5d0e2cc1be"

type Episode struct {
	Id int64
	Length float64
	Path string
	Name string
	Added time.Time
	Deleted bool
	// Description string
	Part int64
	Season int64
	// maybe move these out later
	Offset float64
	Watched bool
	Watched_date time.Time
	SeriesId int
	Series Series
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

type Directory struct {
	Path string `gorm:"primaryKey"`
	Mtime time.Time
	// We need save the directories we saw last time
}

type Config struct {
	Mediapath string
	Dbpath string
	Treshhold float64
	LuaPluginPath string
}

func db_get_pathinfo(db *gorm.DB, path string) (time.Time, bool) {
	var directory Directory
	err := db.Where("path = ?", path).Limit(1).Find(&directory).Error
	if err != nil {
		return directory.Mtime, false
	}
	return directory.Mtime, true
}

func get_filelength(path string) float64 {
	a, err := ffmpeg.Probe(path)
	if err != nil {
		panic(err)
	}
	totalDuration := gjson.Get(a, "format.duration").Float()
	return totalDuration
}

func almostEqual(a, b, threshold float64) bool {
	max := math.Max(a,b)
	min := math.Min(a,b)
    return min/max >= threshold
}
func episode_almostEqual(a float64, bs []int, threshold float64) bool {
	for _, b := range bs {
		if almostEqual(a, float64(b), threshold) {
			return true
		}
	}
	return false
}

// this doesn't work very well with deletion etc
func update_watched(db *gorm.DB, config *Config, path string, watched_amount float64) {
	var episode Episode
	err := db.First(&episode, "path = ?", path).Error
	if err != nil {
		fmt.Printf("File isn't database: %s\n", path)
		return 
	}
	// if we watch more than trashhold
	if (watched_amount / episode.Length) > config.Treshhold {
		episode.Offset = 0
		episode.Watched = true
		episode.Watched_date = time.Now()
	} else {
		episode.Offset = watched_amount
	}
	db.Save(&episode)
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
	id := search.Results[0].ID
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

func get_episode(path string, series *Series, season, episodenum int64) (*Episode, error) {
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
	if episode_almostEqual(length / 60, show_details.EpisodeRunTime, 0.85) {
		episode := &Episode{
			Id:      episode_details.ID,
			Path:    path,
			Length:  length,
			Name:    show_details.Name,
			Added:   time.Now(),
			Deleted: false,
			Part:    episodenum,
			Season:  season,
			Offset:  0,
			Watched: false,
			SeriesId: series.Id,
			Series: *series,
		}
		return episode, nil
	}
	return nil, errors.New("File length doesn't match tmdb file length")
}

type FileInfo struct {
	Title string
	Year int
	Screen_size string
	Source string
	Other string
	Codec string
	Release_group string
	Season int64
	Episode int64
	Group string
	Mimetype string
	Type string
}

// this might be slow but it works for now
func myguessit(path string) (*FileInfo, error) {
	// port it later
	out, err := exec.Command("guessit", "-j", path).Output();
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var fileinfo FileInfo
	if err := json.Unmarshal(out, &fileinfo); err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &fileinfo, nil
}

// we need a helper function
func walker(db *gorm.DB, path string) {
	stat, err := os.Stat(path)
	if err != nil {
		fmt.Printf("Bad path: %s\n", path)
		return
	}
	updatetime := stat.ModTime()
	last_update, entry := db_get_pathinfo(db, path)
	if entry && !last_update.Before(updatetime) {
		return
	}
	diretory := &Directory{
		Path: path,
		Mtime: updatetime,
	}
	if entry {
		db.Create(diretory)
	} else {
		db.Save(diretory)
	}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if f.Name()[0] == '.' {
			continue
		}
		newpath := filepath.Join(path, f.Name())
		stat, _ := os.Stat(newpath)
		if stat.IsDir() {
			walker(db, newpath)
		} else {
			// doesn't work atm
			guessed, err := myguessit(f.Name())
			if err != nil {
				continue
			}
			if len(guessed.Mimetype) < 5 || guessed.Mimetype[:5] != "video" {
				continue
			}
			series, err := get_series(guessed.Title)
			if err != nil {
				continue
			}
			episode, err := get_episode(newpath, series, guessed.Season, guessed.Episode)
			if err != nil {
				continue
			}
			// XXX do we need to make this updateadble
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
			db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
		}
	}
}

func Conf() Config {
	viper.SetConfigName("config")
	viper.AddConfigPath(xdg.ConfigHome + "/movix/")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	viper.SetDefault("Treshhold", 0.9)
	viper.SetDefault("LuaPluginPath", "/home/dagle/code/govix/movix.lua")
	var conf Config

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	err := viper.Unmarshal(&conf)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
	}
	return conf
}

func useage() {
	fmt.Fprintf(os.Stderr, "usage: movix mode [args]\n")
    flag.PrintDefaults()
    os.Exit(2)
}

func watchusage() {
	fmt.Fprintf(os.Stderr, "usage: movix watched path float\n")
    flag.PrintDefaults()
    os.Exit(2)
}

// can we pass in a string?
func get_next(db *gorm.DB) (*Episode, error) {
	var episode Episode
	// Might be correct, first try?
	err := db.Joins("Series").
		Where("watched = ? and Series__subscribed = ?", false, true).
		Order("Series__last_watched, season, part").
		First(&episode).
		Error
	if err != nil {
		panic("sql error")
	}
	return &episode, nil
}

// we only want the first of each series
func get_nexts(db *gorm.DB) ([]Episode, error) {
	var episodes []Episode
	err := db.Joins("Series").
		Where("watched = ? and Series__subscribed = ?", false, true).
		Order("Series__last_watched, season, part").
		Group("Series.id").
		Find(&episodes).
		Error
	if err != nil {
		panic("sql error")
	}
	return episodes, nil
}

func main() {
	flag.Usage = useage
	flag.String("search", "", "A search to filter out")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		useage()
	}

	conf := Conf()
	db, err := gorm.Open(sqlite.Open(conf.Dbpath), &gorm.Config{
		// Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		panic("Couldn't open movix database")
	}
	switch args[0] {
	case "create":
	case "new":
		db.AutoMigrate(&Directory{})
		db.AutoMigrate(&Episode{})
		db.AutoMigrate(&Series{})
		walker(db, conf.Mediapath)
	case "watched":
		if len(args) < 3 {
			watchusage()
		}
		s, err := strconv.ParseFloat(args[2], 32); 
		if err != nil {
			fmt.Println(err)
			watchusage()
		}
		// should be able to take an id?
		update_watched(db, &conf, args[1], s)
	case "next":
		entry, _ := get_next(db)
		fmt.Printf("offset: %d\n", entry.Offset)
		err := exec.Command("mpv",
			"--script=" + conf.LuaPluginPath, 
			"--start=" + fmt.Sprintf("%f", entry.Offset),
			entry.Path).Run()
		// err := exec.Command("mpv", entry.Path).Run()
		if err != nil {
			fmt.Println(err)
		}

		// fmt.Printf("%s\n",entry.Path)
	case "nexts":
		entries, _ := get_nexts(db)
		for _, e := range entries {
			fmt.Printf("%s\n", e.Series.Name)
		}
	}
}
