package movix

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"time"

	"database/sql"
	"os/exec"
	"path/filepath"

	"github.com/tidwall/gjson"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type Walker struct {
	category  string
	mimetype  string
	filetypes []string
}

type Producer interface {
	Add(db *sql.DB, runtime *Runtime, path string, info *FileInfo) error
	Next(db *sql.DB) ([]string, error)
	Select(db *sql.DB, name string) (*Entry, error)
}

type URIProducer interface {
	Producer
	Match(uri string) bool
}

type FsProducer interface {
	Producer
	FsMatch(info *FileInfo) bool
}

// This should be iterators etc instead
type Seekable interface {
	skipUntil(db *sql.DB, name string, season, episode int64) error
	unmarkAfter(db *sql.DB, name string, season, episode int64) error
}

type Runtime struct {
	LuaPluginPath string  // path to the mpv plugin
	Dbpath        string  // path to the database
	Treshhold     float64 // How much do we need to watch before it counts as watching it all.
	MatchLength   float64 // When verifying length, how close does it need to be?
	VerifyLength  bool    // should we verify that the length is correct? Doesn't work atm.
	Rewind        float64 // amount we should rewind when we resume
}

type FileInfo struct {
	Title         string
	Year          int
	Screen_size   string
	Source        string
	Other         string
	Container     string
	Release_group string
	Season        int64
	Episode       int64
	Group         string
	Mimetype      string
	Type          string
}

type Entry struct {
	Id           string
	Length       float64
	Path         string
	Name         string
	Added        int64
	Offset       float64
	Deleted      bool
	Watched      bool
	Watched_date int64
}

// TODO: Make path primary key
func Init(db *sql.DB) error {
	sqlStmt :=
		`create table entry (
		  id text not null primary key,
		  name text not null,
		  length int not null,
		  path text not null,
		  added int not null,
		  offset real,
		  deleted int not null,
		  watched int,
		  watched_date int)
    `
	_, err := db.Exec(sqlStmt)
	return err
}

func (entry *Entry) String() string {
	return entry.Name
}

func (entry *Entry) Save(db *sql.DB) (sql.Result, error) {
	movie_stmt, err := db.Prepare(`insert or replace into entry(
		id, length, path, name, added, offset, deleted, watched, watched_date)
		values(?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	fmt.Println(err)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer movie_stmt.Close()

	result, err := movie_stmt.Exec(entry.Id, entry.Length, entry.Path, entry.Name, entry.Added, entry.Offset,
		entry.Deleted, entry.Watched, entry.Watched_date)

	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return result, err
}

func matchOne(sufflix string, sufflixes []string) bool {
	for _, s := range sufflixes {
		if s == sufflix {
			return true
		}
	}
	return false
}

func GetNext(db *sql.DB, search string, producers ...Producer) (*Entry, error) {
	for _, p := range producers {
		entry, err := p.Select(db, search)
		if err == nil && entry != nil {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("no matching content")
}

func RunWalkers(db *sql.DB, runtime *Runtime, path string, walkers ...FsProducer) error {
	fmt.Printf("Scanning path: %s\n", path)
	abspath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return filepath.WalkDir(abspath, func(path string, d fs.DirEntry, fileerr error) error {
		if d.IsDir() {
			return nil
		}
		guessed, err := myguessit(path)
		if err != nil {
			return err
		}
		fmt.Println(guessed)
		for _, walker := range walkers {
			if walker.FsMatch(guessed) {
				err = walker.Add(db, runtime, path, guessed)
				if err == nil {
					Log("Added file %s to database\n", path)
					return nil
				}
			}
		}
		// we return the last error if we couldn't add the file
		return err
	})
}

func GetEntryPath(db *sql.DB, path string) (*Entry, error) {
	stmt, err := db.Prepare(`select entry.id, 
		entry.length,
		entry.path,
		entry.name,
		entry.added,
		entry.offset,
		entry.deleted,
		entry.watched,
		entry.watched_date
	from entry 
		where path = ?
	`)

	if err != nil {
		return nil, err
	}

	defer stmt.Close()
	var entry Entry
	err = stmt.QueryRow(path).Scan(&entry.Id, &entry.Length, &entry.Path, &entry.Name,
		&entry.Added, &entry.Offset, &entry.Deleted, &entry.Watched, &entry.Watched_date)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// make add another arg, where ? = ?
func GetEntry(db *sql.DB, eid string) (*Entry, error) {
	stmt, err := db.Prepare(`select entry.id, 
		entry.length,
		entry.path,
		entry.name,
		entry.added,
		entry.offset,
		entry.deleted,
		entry.watched,
		entry.watched_date
	from entry 
		where id = ?
	`)

	if err != nil {
		return nil, err
	}

	defer stmt.Close()
	var entry Entry
	err = stmt.QueryRow(eid).Scan(&entry.Id, &entry.Length, &entry.Path, &entry.Name,
		&entry.Added, &entry.Offset, &entry.Deleted, &entry.Watched, &entry.Watched_date)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func UpdateWatched(db *sql.DB, runtime *Runtime, path string, watched_amount float64, full bool) error {
	entry, err := GetEntryPath(db, path)

	if err != nil {
		fmt.Println("Cant update watched: ", err)
		return err
	}
	if full || ((watched_amount / entry.Length) > runtime.Treshhold) {
		entry.Offset = 0
		entry.Watched = true
		entry.Watched_date = time.Now().Unix()
		Log("Watched all of %s\n", path)
	} else {
		entry.Offset = watched_amount
		Log("Watched %f of %s\n", watched_amount, path)
	}
	entry.Save(db)
	return nil
}

func SkipUntil(db *sql.DB, name string, season int64, episode int64, seekable ...Seekable) {
	for _, seek := range seekable {
		err := seek.skipUntil(db, name, season, episode)
		if err == nil {
			Log("Skipped up to s%de%d of %s\n", season, episode, name)
		}
	}
}

func myguessit(path string) (*FileInfo, error) {
	_, file := filepath.Split(path)
	out, err := exec.Command("guessit", "-j", file).Output()
	if err != nil {
		return nil, err
	}
	var fileinfo FileInfo
	if err := json.Unmarshal(out, &fileinfo); err != nil {
		return nil, err
	}
	return &fileinfo, nil
}

func get_filelength(path string) float64 {
	a, err := ffmpeg.Probe(path)
	if err != nil {
		panic(err)
	}
	totalDuration := gjson.Get(a, "format.duration").Float()
	return totalDuration
}

func fsmatch(info *FileInfo) bool {
	filetypes := []string{"mkv", "avi", "mp4", "mov", "wmv", "peg"}
	if !matchOne(info.Container, filetypes) {
		return false
	}
	if info.Mimetype[:5] == "video" && info.Type == "movie" {
		return true
	}
	return false
}

func DeleteWalker(db *sql.DB, root string) error {
	abspath, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	err = filepath.WalkDir(abspath, func(path string, d fs.DirEntry, fileerr error) error {
		if d.IsDir() {
			return nil
		}
		guessed, err := myguessit(path)
		if err != nil {
			return err
		}
		if !fsmatch(guessed) {
			return nil
		}
		entry, err := GetEntryPath(db, path)
		if err != nil {
			log.Println("Couldn't delete entry ", path, "with error: ", err)
			return nil
		}
		entry.Deleted = true
		entry.Save(db)
		return nil
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(abspath)
}

func Delete(db *sql.DB, ids []string) error {
	for _, id := range ids {
		entry, err := GetEntry(db, id)
		if err != nil {
			return nil
		}
		entry.Deleted = true
		entry.Save(db)
	}
	return nil
}

func almostEqual(a, b, threshold float64) bool {
	max := math.Max(a, b)
	min := math.Min(a, b)
	return min/max >= threshold
}
