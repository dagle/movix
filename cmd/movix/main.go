package main

import (
	"bytes"
	"flag"
	"fmt"

	"os"
	"os/exec"
	"strconv"

	"github.com/adrg/xdg"
	"github.com/dagle/movix"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var LuaPath string

type Config struct {
	movix.Runtime
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
	viper.SetDefault("LuaPluginPath", LuaPath + "/movix.lua")
	viper.SetDefault("Perm", 664)
	viper.SetDefault("Mediapath", xdg.UserDirs.Videos + "/movix")
	var conf Config

	if err := viper.ReadInConfig(); err != nil {
		viper.ReadConfig(bytes.NewBuffer([]byte("")))
	}

	err := viper.Unmarshal(&conf.Runtime)
	if err != nil {
		movix.Fatal("Unable to decode into struct: \n", err)
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
	movix.Eprintf("usage: movix mode [args]\n")
    flag.PrintDefaults()
    os.Exit(2)
}

func watchusage() {
	movix.Eprintf("usage: movix watched path float\n")
    flag.PrintDefaults()
    os.Exit(2)
}

func deleteusage() {
	movix.Eprintf("usage: movix delete [paths]\n")
    flag.PrintDefaults()
    os.Exit(2)
}
func suggestusage() {
	movix.Eprintf("usage: movix suggestdel\n")
    flag.PrintDefaults()
    os.Exit(2)
}

func play (conf *Config, entry *movix.Entry) error {
	err := exec.Command("mpv",
		"--script=" + conf.LuaPluginPath, 
		"--start=" + fmt.Sprintf("%f", entry.Offset),
		entry.Path).Run()
	if err != nil {
		return err
	}
	return nil
}

func make_defconf(dirpath string){
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

func GetSuggest(db *gorm.DB, producers ...movix.Producer) {
	for _, producer := range producers {
		res, err := producer.Next(db)
		if err != nil {
			movix.Eprintf("Suggest error: %v", err)
		}
		for _, str := range res {
			fmt.Println(str)
		}
	}
}

func main() {
	flag.Usage = useage

	// check these later
	defpath, _ := xdg.ConfigFile("movix")
	deflog, _ := xdg.StateFile("movix.log")

	flag.StringVar(&logpath, "l", deflog, "File to use for logging")
	flag.BoolVar(&sqllog, "q", false, "Turn on sqllogging")
	flag.StringVar(&dbpath, "d", "","Specify database directory")
	flag.StringVar(&confpath, "c", defpath, "Specify database directory")
	flag.Parse()

	err := movix.LogInit(logpath)
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log: %s\n", err)
	}

	args := flag.Args()
	if len(args) < 1 {
		useage()
	}

	var gormcfg gorm.Config
	if sqllog {
		gormcfg = gorm.Config{Logger: logger.Default.LogMode(logger.Info)}
	} else {
		gormcfg = gorm.Config{}
	}

	conf := Conf(confpath, dbpath)
	db, err := gorm.Open(sqlite.Open(conf.Dbpath), &gormcfg)

	if err != nil {
		movix.Fatal("Couldn't open movix database")
	}

	tv := movix.Make_series()
	movies := movix.Make_movies()

	switch args[0] {
	case "init":
		db.AutoMigrate(&movix.Episode{})
		db.AutoMigrate(&movix.Series{})
		db.AutoMigrate(&movix.Movie{})
		// movix.Log("Database created\n")
	case "add":
		var producers []movix.URIProducer
		if len(args) != 2 {
			movix.Fatal("Add needs a filepath")
		}
		for _, prod := range producers {
			if prod.Match(args[1]) {
				prod.Add(db, &conf.Runtime, args[1], nil)
				return
			}
		}
		movix.RunWalkers(db, &conf.Runtime, args[1], movies, tv)
	case "migrate":
		db.AutoMigrate(&movix.Episode{})
		db.AutoMigrate(&movix.Series{})
		db.AutoMigrate(&movix.Movie{})
	// case "rescan":
	// 	movix.Rescan(db, &conf.Runtime, movies, tv)
	case "suggestdel":
		if len(args) < 2 {
			suggestusage()
		}
		suggest, err := movix.Suggest_deletions(db)
		if err != nil {
			fmt.Println(err)
			suggestusage()
		}
		fmt.Println(suggest)
	case "delete_group":
		if len(args) < 2 {
			deleteusage()
		}
		movix.Delete_group(db, args[1:])
	case "delete":
		if len(args) < 2 {
			deleteusage()
		}
		movix.Delete(db, args[1:])
	case "watched":
		if len(args) < 3 {
			watchusage()
		}
		if args[2] == "full" {
			movix.UpdateWatched(db, &conf.Runtime, args[1], 0, true)
		} else {
			s, err := strconv.ParseFloat(args[2], 32); 
			if err != nil {
				movix.Fatal("Watched needs a value or full: %v", err)
			}
			movix.UpdateWatched(db, &conf.Runtime, args[1], s, false)
		}
	case "next":
		search := ""
		if len(args) != 2 || args[1] == "" {
			movix.Fatal("movix next needs a title argument")
		} 
		search  = args[1]
		entry, err := movix.GetNext(db, search, movies, tv)
		if err == nil {
			play(conf, entry)
		}
	case "suggest":
		GetSuggest(db, tv, movies)
	case "skip":
		if len(args) != 4 {
			movix.Fatal("Skip needs 3 arguments: show season episode")
		}
		season, err := strconv.Atoi(args[2])
		episode, err2 := strconv.Atoi(args[3])
		if err != nil || err2 != nil {
			movix.Fatal("Arguments 3 and 4 needs to be integers")
		}
		movix.SkipUntil(db, args[1], int64(season), int64(episode), tv)
	case "version":
		fmt.Printf("Movix version: %s\n", "0.0.1")
	default:
		useage()
	}
}
