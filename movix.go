package movix

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tidwall/gjson"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"gorm.io/gorm"
)

type Walker struct {
	category string
	mimetype string
	filetypes []string
}

type Producer interface {
	Add(db *gorm.DB, runtime *Runtime, path string, info *FileInfo) error
	Next(db *gorm.DB) ([]string, error)
	Select(db *gorm.DB, name string) (*Entry, error)
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
	skipUntil(db *gorm.DB, name string, season, episode int64) error
	unmarkAfter(db *gorm.DB, name string, season, episode int64) error
}

type Runtime struct {
	Mediapath string
	Perm fs.FileMode
	Dbpath string
	Treshhold float64
	MatchLength float64
	Move bool
	VerifyLength bool
}

type FileInfo struct {
	Title string
	Year int
	Screen_size string
	Source string
	Other string
	Container string
	Release_group string
	Season int64
	Episode int64
	Group string
	Mimetype string
	Type string
}

type Entry struct {
	Id int64
	// gorm.Model
	// RemoteId: int64,
	Length float64
	Path string
	Name string
	Added int64
	Offset float64
	Deleted bool
	Watched bool
	Watched_date int64
}

type Deleteable struct {
	Path string
	Rank float64
}

func (entry *Entry) String() string{
	return entry.Name
}

func matchOne(sufflix string, sufflixes []string) bool {
	for _, s := range sufflixes {
		if s == sufflix {
			return true
		}
	}
	return false
}

func GetNext(db *gorm.DB, search string, producers ...Producer) (*Entry, error) {
	for _, p := range producers {
		entry, err := p.Select(db, search)
		if err != nil && entry != nil {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("no matching content")
}

func RunWalkers(db *gorm.DB, runtime *Runtime, path string, walkers ...FsProducer) error {
	return filepath.WalkDir(path, func(path string, d fs.DirEntry, fileerr error) error {
		if d.IsDir() {
			return nil
		}
		guessed, err := myguessit(path)
		if err != nil {
			return nil
		}
		for _, walker := range walkers {
			if walker.FsMatch(guessed) {
				err := walker.Add(db, runtime, path, guessed)
				if err != nil {
					log.Printf("Failed adding matched content: %s", path)
				}
				return err
			}
		}
		return nil
	})
}

// TODO
func Rescan(db *gorm.DB, runtime *Runtime, walkers ...FsProducer) error {
	return filepath.WalkDir(runtime.Mediapath, func(path string, 
		d fs.DirEntry, fileerr error) error {
			// get a list of all matches
			// get a list of all entries in db
			return nil
	})
}

func UpdateWatched(db *gorm.DB, runtime *Runtime, path string, watched_amount float64, full bool) error {
	var entry Entry 
	err := db.First(&entry, "path = ?", path).Error
	if err != nil {
		return err
	}
	// if we watch more than trashhold
	if full || ((watched_amount / entry.Length) > runtime.Treshhold) {
		entry.Offset = 0
		entry.Watched = true
		entry.Watched_date = time.Now().Unix()
	} else {
		entry.Offset = watched_amount
	}
	db.Save(&entry)
	return nil
}
func SkipUntil(db *gorm.DB, name string, season int64, episode int64, seekable ...Seekable) error {
	for _, seek := range seekable {
		err := seek.skipUntil(db, name, season, episode)
		if err != nil {
			return err
		}
	}
	return nil
}

func myguessit(path string) (*FileInfo, error) {
	_, file := filepath.Split(path)
	out, err := exec.Command("guessit", "-j", file).Output();
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

// How do we handle criteria?
// What do we return?
// "How long haven't I watched this"

// idle-time
// unixepoch-watched_date, 6 months => 1000 points

// left-time TODO
// normalize -(offset/length)

// Burst-time TODO
// nomalize -(date-watch_date) 
func Suggest_deletions(db *gorm.DB) ([]string, error) {
	ret := []string{}
	var entries []Entry
	past := time.Now().AddDate(0, -6, 0).Unix()
	err := db.Where("watched_date >= ? or (watched = ? and deleted = ?)", past, false, true).
		Order("watched_date").
		Find(&entries).
		Error
	if err != nil {
		return nil, err
	}
	return ret, nil
}
func Suggest_groups(db *gorm.DB) ([]string, error) {
	var episodes []Episode
	err := db.Joins("Series").Joins("Entry").
		Where("watched = ? and deleted = ?", true, false).
		Order("last_watched, season, part").
		// Distinct("Series.id").
		Group("Series.id").
		Find(&episodes).
		Error

	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range episodes {
		names = append(names, e.Series.Name)
	}
	return names, nil
}

func Delete_group(db *gorm.DB, names []string) error {
	var entries []Entry
	for _, name := range(names) {
		err := db.Joins("Series").Joins("Episode").
		Where("deleted = ? and Series.name = ?", false, name).
		Find(&entries).
		Error
		if err != nil {
			return err
		}
		for _, entry := range(entries) {
			if err := os.Remove(entry.Path); err != nil {
				return err
			}
			entry.Deleted = true;
			db.Save(&entry)
		}
	}
	return nil
}

func Delete(db *gorm.DB, paths []string) error {
	for _, path := range(paths) {
		var entry Entry
		err := db.Where("path = ?", path).First(&entry).Error
		if err != nil {
			return err
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		entry.Deleted = true;
		db.Save(&entry)
	}
	return nil
}

func almostEqual(a, b, threshold float64) bool {
	max := math.Max(a,b)
	min := math.Min(a,b)
    return min/max >= threshold
}
