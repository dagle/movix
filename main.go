package main

import (
	"bytes"
	"flag"
	"fmt"

	"os"
	"os/exec"
	"strconv"

	"database/sql"
  _ "github.com/mattn/go-sqlite3"
	"github.com/adrg/xdg"
	backend "github.com/dagle/movix/backend"

	"github.com/spf13/viper"
)

var LuaPath string

type Config struct {
	backend.Runtime
}

func Conf(configpath, dbpath string) *Config {
	viper.SetConfigName("config")
	viper.AddConfigPath(configpath)
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	viper.SetDefault("Treshhold", 0.9)
	viper.SetDefault("MatchLength", 0.35)
	viper.SetDefault("VerifyLength", true)
	viper.SetDefault("Moved", true)
	viper.SetDefault("LuaPluginPath", LuaPath+"/movix.lua")
	viper.SetDefault("Perm", 664)
	viper.SetDefault("Mediapath", xdg.UserDirs.Videos+"/movix")
	var conf Config

	if err := viper.ReadInConfig(); err != nil {
		viper.ReadConfig(bytes.NewBuffer([]byte("")))
	}

	err := viper.Unmarshal(&conf.Runtime)
	if err != nil {
		backend.Fatal("Unable to decode into struct: \n", err)
	}

	if dbpath != "" {
		conf.Dbpath = dbpath
	}
	if conf.Dbpath == "" {
		defdbpath, _ := xdg.DataFile("movix.db")
		conf.Dbpath = defdbpath
	}

	return &conf
}

func useage() {
	backend.Eprintf("usage: movix mode [args]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func watchusage() {
	backend.Eprintf("usage: movix watched path float\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func deleteusage() {
	backend.Eprintf("usage: backend delete [paths]\n")
	flag.PrintDefaults()
	os.Exit(2)
}
func suggestusage() {
	backend.Eprintf("usage: backend suggestdel\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func play(conf *Config, entry *backend.Entry) error {
	err := exec.Command("mpv",
		"--script="+conf.LuaPluginPath,
		"--start="+fmt.Sprintf("%f", entry.Offset),
		entry.Path).Run()
	if err != nil {
		return err
	}
	return nil
}

func make_defconf(dirpath string) {
	// reader := bufio.NewReader(os.Stdin)
	// fmt.Print("Path to media library: ")
	viper.WriteConfig()
	// text, _ := reader.ReadString('\n')
	// fmt.Print(text)
}

var logpath string
var sqllog bool
var dbpath string
var confpath string

func GetSuggest(db *sql.DB, producers ...backend.Producer) {
	for _, producer := range producers {
		res, err := producer.Next(db)
		if err != nil {
			backend.Eprintf("Suggest error: %v", err)
		}
		for _, str := range res {
			fmt.Println(str)
		}
	}
}

func main() {
	flag.Usage = useage

	// check these later
	defpath, _ := xdg.ConfigFile("backend")
	deflog, _ := xdg.StateFile("backend.log")

	flag.StringVar(&logpath, "l", deflog, "File to use for logging")
	flag.BoolVar(&sqllog, "q", false, "Turn on sqllogging")
	flag.StringVar(&dbpath, "d", "", "Specify database directory")
	flag.StringVar(&confpath, "c", defpath, "Specify database directory")
	flag.Parse()

	err := backend.LogInit(logpath)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log: %s\n", err)
	}

	args := flag.Args()
	if len(args) < 1 {
		useage()
	}

	// var gormcfg gorm.Config
	// if sqllog {
	// 	gormcfg = gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	// } else {
	// 	gormcfg = gorm.Config{}
	// }

	conf := Conf(confpath, dbpath)
	db, err := sql.Open("sqlite3", "./foo.db")
	// db, err := gorm.Open(sqlite.Open(conf.Dbpath), &gormcfg)

	if err != nil {
		backend.Fatal("Couldn't open backend database")
	}

	tv := backend.Make_series()
	movies := backend.Make_movies()

	switch args[0] {
	case "init":
		// db.AutoMigrate(&backend.Episode{})
		// db.AutoMigrate(&backend.Series{})
		// db.AutoMigrate(&backend.Movie{})
		// backend.Log("Database created\n")
	case "add":
		var producers []backend.URIProducer
		if len(args) != 2 {
			backend.Fatal("Add needs a filepath")
		}
		for _, prod := range producers {
			if prod.Match(args[1]) {
				prod.Add(db, &conf.Runtime, args[1], nil)
				return
			}
		}
		// backend.RunWalkers(db, &conf.Runtime, args[1], movies, tv)
	case "migrate":
		// db.AutoMigrate(&backend.Episode{})
		// db.AutoMigrate(&backend.Series{})
		// db.AutoMigrate(&backend.Movie{})
	// case "rescan":
	// 	backend.Rescan(db, &conf.Runtime, movies, tv)
	case "suggestdel":
		if len(args) < 2 {
			suggestusage()
		}
		suggest, err := backend.Suggest_deletions(db)
		if err != nil {
			fmt.Println(err)
			suggestusage()
		}
		fmt.Println(suggest)
	case "delete_group":
		if len(args) < 2 {
			deleteusage()
		}
		backend.Delete_group(db, args[1:])
	case "delete":
		if len(args) < 2 {
			deleteusage()
		}
		backend.Delete(db, args[1:])
	case "watched":
		if len(args) < 3 {
			watchusage()
		}
		if args[2] == "full" {
			backend.UpdateWatched(db, &conf.Runtime, args[1], 0, true)
		} else {
			s, err := strconv.ParseFloat(args[2], 32)
			if err != nil {
				backend.Fatal("Watched needs a value or full: %v", err)
			}
			backend.UpdateWatched(db, &conf.Runtime, args[1], s, false)
		}
	case "next":
		search := ""
		if len(args) != 2 || args[1] == "" {
			backend.Fatal("backend next needs a title argument")
		}
		search = args[1]
		entry, err := backend.GetNext(db, search, movies, tv)
		if err == nil {
			play(conf, entry)
		}
	case "suggest":
		GetSuggest(db, tv, movies)
	case "skip":
		if len(args) != 4 {
			backend.Fatal("Skip needs 3 arguments: show season episode")
		}
		season, err := strconv.Atoi(args[2])
		episode, err2 := strconv.Atoi(args[3])
		if err != nil || err2 != nil {
			backend.Fatal("Arguments 3 and 4 needs to be integers")
		}
		backend.SkipUntil(db, args[1], int64(season), int64(episode), tv)
	case "version":
		fmt.Printf("backend version: %s\n", "0.0.1")
	default:
		useage()
	}
}
