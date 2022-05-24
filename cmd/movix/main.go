package main

import (
	"bufio"
	"flag"
	"fmt"

	"log"
	"os"
	"os/exec"
	"strconv"

	"github.com/adrg/xdg"
	"github.com/spf13/viper"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"github.com/dagle/movix"
)

var LuaPath string

type Config struct {
	movix.Runtime
	LuaPluginPath string
}

func Conf(configpath, dbpath string) *Config {
	viper.SetConfigName("config")
	viper.AddConfigPath(configpath)
	viper.AutomaticEnv()
	viper.SetConfigType("yml")
	viper.SetDefault("Treshhold", 0.9)
	viper.SetDefault("MatchLength", 0.85)
	viper.SetDefault("VerifyLength", true)
	viper.SetDefault("Moved", true)
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

func make_config(dirpath string, conf *Config){
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Path to media library: ")
	text, _ := reader.ReadString('\n')
	// make a config and write it or just
	// Mediapath: "/home/dagle/download/media/"?
	fmt.Print(text)
}

var logging bool
var sqllog bool
var dbpath string
var confpath string

func GetNexts(db *gorm.DB, producers ...movix.Producer) {
	for _, producer := range producers {
		res, err := producer.Next(db)
		if err != nil {
			log.Printf("Nexts error: %s", err)
		}
		for _, str := range res {
			fmt.Println(str)
		}
	}
}

func main() {
	flag.Usage = useage
	defpath, _ := xdg.ConfigFile("movix/config")
	flag.BoolVar(&logging, "l", false, "Turn on logging")
	flag.BoolVar(&sqllog, "q", false, "Turn on sqllogging")
	flag.StringVar(&dbpath, "d", "", "Specify database directory")
	flag.StringVar(&confpath, "c", defpath, "Specify database directory")
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

	tv := movix.Make_series()
	movies := movix.Make_movies()

	switch args[0] {
	case "init":
		make_config(confpath, conf)
	case "add":
		var producers []movix.URIProducer

		if len(args) != 2 {
			panic("Add needs a filepath")
		}
		for _, prod := range producers {
			if prod.Match(args[1]) {
				prod.Add(db, &conf.Runtime, args[1], nil)
				return
			}
		}
		movix.RunWalkers(db, &conf.Runtime, args[1], movies, tv)
	case "rescan": // maybe call this migrate?
		db.AutoMigrate(&movix.Episode{})
		db.AutoMigrate(&movix.Series{})
		db.AutoMigrate(&movix.Movie{})
		fmt.Printf("Media path: %s\n", conf.Mediapath)
	case "watched":
		if len(args) < 3 {
			watchusage()
		}
		if args[2] == "full" {
			movix.UpdateWatched(db, &conf.Runtime, args[1], 0, true)
		} else {
			s, err := strconv.ParseFloat(args[2], 32); 
			if err != nil {
				fmt.Println(err)
				watchusage()
			}
			movix.UpdateWatched(db, &conf.Runtime, args[1], s, false)
		}
	case "next":

		search := ""
		if len(args) != 2 || args[1] == "" {
			log.Fatal("movix next needs a title argument")
		} 
		search  = args[1]
		entry, err := movix.GetNext(db, search, movies, tv)
		if err == nil {
			play(conf, entry)
		}
	case "nexts":
		GetNexts(db, tv, movies)
	case "skip":
		if len(args) != 4 {
			log.Fatal("Skip needs 3 arguments: show season episode")
		}
		season, err := strconv.Atoi(args[2])
		episode, err2 := strconv.Atoi(args[3])
		if err != nil || err2 != nil {
			log.Fatal("Arguments 3 and 4 needs to be integers")
		}
		movix.SkipUntil(db, args[1], season, episode, movix.Make_series())
	}
}
