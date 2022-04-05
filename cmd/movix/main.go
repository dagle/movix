package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"

	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var LuaPath string
// Add this to the makefile etc
var APIKEY string = "e130a499c97798cfac3ffb5d0e2cc1be"

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
	// Kind String can we use an enum?
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
	Perm fs.FileMode
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

// check 
func walker(db *gorm.DB, conf *Config, path string) {
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
			walker(db, conf, newpath)
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
				movie.Move(conf, guessed.Codec)
				
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(movie)
			case "episode":
				series, err := get_series(guessed.Title)
				if err != nil {
					continue
				}
				episode, err := series.get_episode(newpath, guessed.Season, guessed.Episode)
				if err != nil {
					continue
				}
				episode.Move(conf, guessed.Codec)
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(series)
				db.Clauses(clause.OnConflict{DoNothing: true}).Create(episode)
			}
		}
	}
}

// func del(db *gorm.DB, path string) {
// 	var entry Entry 
// 	if err := db.First(&entry, "path = ?", path).Error; err == nil {
// 		entry.Deleted = true
// 		db.Save(&entry)
// 		if filesystem {
// 			os.Remove(path)
// 		}
// 	}
// }

// We set thing from the Makefile

func Conf(configpath, dbpath string) *Config {
	viper.SetConfigName("config")
	viper.AddConfigPath(configpath)
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	viper.SetDefault("Treshhold", 0.9)
	viper.SetDefault("LuaPluginPath", LuaPath + "/movix.lua")
	viper.SetDefault("Perm", 664)
	viper.SetDefault("Mediapath", xdg.UserDirs.Videos + "/movix")
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
	if dbpath != "" {
		conf.Dbpath = dbpath
	} else if conf.Dbpath == "" {
		conf.Dbpath = conf.Mediapath + "/.movix.db"
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

func make_config(){
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Path to media library: ")
	text, _ := reader.ReadString('\n')
	// make a config and write it or just
	// Mediapath: "/home/dagle/download/media/"?
	fmt.Print(text)
}

var directories = []string{"tv", "movies"}

func add_new(conf *Config) int {
	dbfile, err := os.Stat(conf.Dbpath)

	if err != nil {
		fmt.Println(err)
	}
	
	dbtime := dbfile.ModTime()
	fmt.Println(dbtime)
	i := 0
	for _, dir := range directories {
		statfile, err := os.Stat(dir)
		if err != nil {
			continue
		}

		if statfile.ModTime().After(dbtime) {
			i++
		}
	}
	
	return i
}

var logging bool
var sqllog bool
var dbpath string
var confpath string

const (
	FS = iota
	YT
)

func protocol(uri string) int {
	return FS
}

func add_youtube(db *gorm.DB, conf *Config, uri string) {
}

func main() {
	flag.Usage = useage
	// flag.String("search", "", "A search to filter out")
	// defpath := xdg.ConfigHome + "/movix/"
	defpath, _ := xdg.ConfigFile("movix/config")
	flag.BoolVar(&logging, "l", false, "Turn on logging")
	flag.BoolVar(&sqllog, "q", false, "Turn on sqllogging")
	flag.StringVar(&dbpath, "d", "", "Specify database directory")
	flag.StringVar(&confpath, "c", defpath, "Specify database directory")
	// flag.BoolVar(&filesystem, "f", true, "Disable filesystem actions")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		useage()
	}

	
	conf := Conf(confpath, dbpath)
	var gormcfg gorm.Config
	if sqllog {
		gormcfg = gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	} else {
		gormcfg = gorm.Config{}
	}
	db, err := gorm.Open(sqlite.Open(conf.Dbpath), &gormcfg)
	if err != nil {
		panic("Couldn't open movix database")
	}
	switch args[0] {
	case "config":
		make_config()
	case "add":
		if len(args) != 2 {
			panic("Add needs a filepath")
		}
		switch protocol(args[1]) {
		case FS:
			walker(db, conf, args[1])
		case YT:
			add_youtube(db, conf, args[1])
		}
	case "new":
		add_new(conf)
	case "rescan": // maybe call this migrate?
		db.AutoMigrate(&Episode{})
		db.AutoMigrate(&Series{})
		db.AutoMigrate(&Movie{})
		fmt.Printf("Media path: %s\n", conf.Mediapath)
		walker(db, conf, conf.Mediapath)
	case "play":
		if len(args) != 2 {
			panic("play needs a filename")
		}
		play_file(db, args[1], conf)
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
			log.Fatal("Skip needs 3 arguments: show season episode")
		}
		season, err := strconv.Atoi(args[2])
		episode, err2 := strconv.Atoi(args[3])
		if err != nil || err2 != nil {
			log.Fatal("Arguments 3 and 4 needs to be integers")
		}
		skipUntil(db, args[1], season, episode)
	}
}
