package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"strings"

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
	"gorm.io/gorm/logger"
)

const APIKEY = "e130a499c97798cfac3ffb5d0e2cc1be"

type Entry struct {
	Id int64
	Length float64
	Path string
	Name string
	Added time.Time
	Offset float64
	Deleted bool
	Watched bool
	Watched_date time.Time
}

type Movie struct {
	gorm.Model
	EntryID int64
	Entry Entry
}

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

type Config struct {
	Mediapath string
	Dbpath string
	Treshhold float64
	LuaPluginPath string
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

func episodeAlmostEqual(a float64, bs []int, threshold float64) bool {
	for _, b := range bs {
		if almostEqual(a, float64(b), threshold) {
			return true
		}
	}
	return false
}

func updateWatched(db *gorm.DB, config *Config, path string, watched_amount float64, full bool) {
	var entry Entry 
	err := db.First(&entry, "path = ?", path).Error
	if err != nil {
		fmt.Printf("File isn't database: %s\n", path)
		return 
	}
	// if we watch more than trashhold
	if full || (watched_amount / entry.Length) > config.Treshhold {
		entry.Offset = 0
		entry.Watched = true
		entry.Watched_date = time.Now()
	} else {
		entry.Offset = watched_amount
	}
	db.Save(&entry)
}

func get_movie(path, title string) (*Movie, error) {
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
	if almostEqual(length / 60, float64(movie_details.Runtime), 0.85) {
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
	if episodeAlmostEqual(length / 60, show_details.EpisodeRunTime, 0.85) {
		episode := &Episode{
			Entry:Entry{
				Id:      episode_details.ID,
				Path:    path,
				Length:  length,
				Name:    show_details.Name,
				Added:   time.Now(),
				Deleted: false,
				Offset:  0,
				Watched: false,
			},
			Part:    episodenum,
			Season:  season,
			SeriesId: series.Id,
			Series: *series,
		}
		return episode, nil
	}
	return nil, errors.New("file length doesn't match tmdb file length")
}

func myguessit(path string) (*FileInfo, error) {
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

var filetype []string = []string{"mkv", "avi", "mp4", "mov", "wmv", "peg"}

func fileExtentison(path string) bool {
	for _, ext := range filetype {
		if len(path) > 3 && path[len(path)-3:] == ext {
			return true
		}
	}
	return false
}

func walker(db *gorm.DB, path string) {
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
			if !fileExtentison(newpath) {
				continue
			}
			guessed, err := myguessit(f.Name())
			if err != nil {
				continue
			}
			// we need suport for things like vods
			if len(guessed.Mimetype) < 5 || guessed.Mimetype[:5] != "video" {
				continue
			}
			log.Printf("Adding file: %s\n", newpath)
			switch guessed.Type {
			case "movie":
				movie, err := get_movie(newpath, guessed.Title)
				if err != nil {
					continue
				}
				// XXX do we need to make this updateadble?
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(movie)
			case "episode":
				series, err := get_series(guessed.Title)
				if err != nil {
					continue
				}
				episode, err := get_episode(newpath, series, guessed.Season, guessed.Episode)
				if err != nil {
					continue
				}
				// XXX do we need to make this updateadble?
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
			}
		}
	}
}

func del(db *gorm.DB, path string) {
	var entry Entry 
	if err := db.First(&entry, "path = ?", path).Error; err == nil {
		entry.Deleted = true
		db.Save(&entry)
		if filesystem {
			os.Remove(path)
		}
	}
}

var LuaPath string

func Conf() *Config {
	viper.SetConfigName("config")
	viper.AddConfigPath(xdg.ConfigHome + "/movix/")
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	viper.SetDefault("Treshhold", 0.9)
	viper.SetDefault("LuaPluginPath", LuaPath + "/movix.lua")
	var conf Config

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file, %s", err)
	}

	err := viper.Unmarshal(&conf)
	if err != nil {
		fmt.Printf("Unable to decode into struct, %v", err)
	}
	stat, err := os.Stat(conf.Mediapath)
	if err != nil {
		log.Fatal(err)
	}
	if !stat.IsDir() {
		log.Fatal(err)
	}
	if conf.Dbpath == "" {
		conf.Dbpath = xdg.ConfigHome + "/movix/" + "movix.db"

	}
	return &conf
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
func get_next(db *gorm.DB, name string) (*Entry, error) {
	var episode Episode
	// querystring := "watched = ? and deleted = ?"
	// if name != "" {
	// }
	err := db.Joins("Series").Joins("Entry").
		Where("watched = ? and deleted = ? and Series.name = ?", false, false, name).
		Order("last_watched, season, part").
		Group("Series.id").
		First(&episode).
		Error
	if err == nil {
		return &episode.Entry, nil
	}
	var entry Entry
	err = db. Where("watched = ? and deleted = ? and name = ?", false, false, name).
		First(&entry).
		Error
	if err == nil {
		return &entry,nil
	}
	return nil, errors.New("no entry found")
}

// do we even want this function? Can't we just do head?
func get_nexts_tv(db *gorm.DB) ([]Episode, error) {
	var episodes []Episode
	err := db.Joins("Series").Joins("Entry").
		Where("watched = ? and subscribed = ? and deleted = ?", false, true, false).
		Order("last_watched, season, part").
		Group("Series.id").
		Find(&episodes).
		Error
	return episodes, err
}

/// We want something more general.
func skipUntil(db *gorm.DB, name string, season, episode int) error {
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

func unmarkAfter(db *gorm.DB, name string, season, episode int) error {
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

func get_nexts_movie(db *gorm.DB) ([]Movie, error) {
	var movies []Movie
	err := db.Joins("Entry").
		Where("watched = ? and deleted = ?", false, false).
		Find(&movies).
		Error
	return movies, err
}

func play_file(db *gorm.DB, path string, conf *Config) {
	var entry Entry
	err := db.First(&entry, "path = ?", path).
		Error
	if err != nil {
		panic("Error exucting sql")
	}
	entry.play(conf)
}

func (entry *Entry) play (conf *Config) {

	err := exec.Command("mpv",
		"--script=" + conf.LuaPluginPath, 
		"--start=" + fmt.Sprintf("%f", entry.Offset),
		entry.Path).Run()
	if err != nil {
		fmt.Println(err)
	}
}
func (entry *Entry) show() {
	fmt.Printf("%s\n", entry.Name)
}

type Mediatype int64

const (
	none Mediatype = iota
	tv 
	movie
)

var errnoMatch error = errors.New("couldn't find a Mediatype")

/// Tries to find media file to determine what kind of content we have 
/// (movie, tv, etc) from the filename. Once it found one, it will report the whole
/// collection of being that.
func findType(path string) (Mediatype, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if f.Name()[0] == '.' {
			continue
		}
		newpath := filepath.Join(path, f.Name())
		stat, err := os.Stat(newpath)
		if err != nil {
			return none, err
		}
		if stat.IsDir() {
			i, e := findType(newpath)
			if e == nil {
				return i, nil
			}
		} else {
			if !fileExtentison(newpath) {
				continue
			}
			guessed, err := myguessit(f.Name())
			if err != nil {
				continue
			}
			// we need suport for things like vods
			if len(guessed.Mimetype) < 5 || guessed.Mimetype[:5] != "video" {
				continue
			}
			switch guessed.Type {
			case "movie":
				return movie, nil 
			case "episode":
				return tv, nil
			}
		}
	}
	return none, errnoMatch
}

func sort(conf *Config, path string) error{
	mt, e := findType(path)
	if e != nil {
		log.Fatal("Can't add file")
	}
	var dir string
	switch mt {
	case tv:
		dir = "tv/"
	case movie:
		dir = "movies/"
	}
	stat, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	return os.Rename(path, conf.Mediapath + dir + stat.Name())
}

func createDirs(conf *Config) {
	// add more to this later
	dirs := []string{"/tv", "/movies"}
	for _, d := range dirs {
		os.Mkdir(conf.Mediapath + d, 0755)
	}
}

var filesystem bool
var logging bool

func main() {
	flag.Usage = useage
	// flag.String("search", "", "A search to filter out")
	sqllog := flag.Bool("q", false, "Log sql queries to output")
	flag.BoolVar(&logging, "l", false, "Turn on logging")
	flag.BoolVar(&filesystem, "f", true, "Disable filesystem actions")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		useage()
	}

	
	conf := Conf()
	var gormcfg gorm.Config
	if *sqllog {
		gormcfg = gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	} else {
		gormcfg = gorm.Config{}
	}
	db, err := gorm.Open(sqlite.Open(conf.Dbpath), &gormcfg)
	if err != nil {
		panic("Couldn't open movix database")
	}
	switch args[0] {
	case "create":
		createDirs(conf)
	case "add":
		if len(args) != 2 {
			panic("Sorts needs a filepath")
		}
		walker(db, args[1])
	case "sort":
		if len(args) != 2 {
			panic("Sorts needs a filepath")
		}
		e := sort(conf, args[1])
		if e != nil {
			log.Fatal(e)
		}
	case "play":
		if len(args) != 2 {
			panic("play needs a filename")
		}
		play_file(db, args[1], conf)
	case "rescan":
		db.AutoMigrate(&Episode{})
		db.AutoMigrate(&Series{})
		db.AutoMigrate(&Movie{})
		fmt.Printf("Media path: %s\n", conf.Mediapath)
		walker(db, conf.Mediapath)
	case "watched":
		if len(args) < 3 {
			watchusage()
		}
		if args[2] == "full" {
			updateWatched(db, conf, args[1], 0, true)
		} else {
			s, err := strconv.ParseFloat(args[2], 32); 
			if err != nil {
				fmt.Println(err)
				watchusage()
			}
			// should be able to take an id?
			updateWatched(db, conf, args[1], s, false)
		}
	case "next":
		search := ""
		if len(args) != 2 || args[1] == "" {
			log.Fatal("movix next needs a title argument")
		} 
		search  = args[1]
		entry, err := get_next(db, search)
		if err == nil {
			entry.play(conf)
		}
	case "nexts":
		episodes, _ := get_nexts_tv(db)
		for _, e := range episodes {
			e.Entry.show()
		}
		movies, _ := get_nexts_movie(db)
		for _, e := range movies {
			e.Entry.show()
		}
	case "skip":
		if len(args) != 4 {
			log.Fatal("Skip needs 3 arguments")
		}
		season, err := strconv.Atoi(args[2])
		episode, err2 := strconv.Atoi(args[3])
		if err != nil || err2 != nil {
			log.Fatal("Arguments 3 and 4 needs to be integers")
		}
		skipUntil(db, args[1], season, episode)
	}
}
