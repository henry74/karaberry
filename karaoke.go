package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"reflect"
	"sync"
	"time"
)

type (
	Song struct {
		ID        int    `json:"id"`
		Artist    string `json:"artist"`
		Name      string `json:"name"`
		YoutubeID string `json:"youtube_id"`
		Filename  string `json:"filename"`
	}

	SongHistory struct {
		songs []*Song
	}

	SongQueue struct {
		q         []*Song
		size      int
		lock      *sync.Mutex
		addlisten chan int
	}

	Karaoke struct {
		history SongHistory
		Queue   chan *Song
	}
)

var (
	songList []*Song
	karaoke  = Karaoke{
		history: SongHistory{songs: []*Song{}},
		Queue:   make(chan *Song, Config.QueueSize),
	}
)

func init() {
	songFilename := Config.MediaFolder + "/songs.csv"
	songFile, openErr := os.Open(songFilename)
	if openErr != nil {
		panic("Could not open CSV file")
	}
	defer songFile.Close()

	reader := csv.NewReader(songFile)
	reader.Comma = '|'
	rows, readErr := reader.ReadAll()
	if readErr != nil {
		panic(readErr)
	}

	for i, row := range rows {
		var filename string
		if row[3] != "" {
			filename = fmt.Sprintf("%s/%s", Config.MediaFolder, row[3])
		}
		songList = append(songList, &Song{i, row[0], row[1], row[2], filename})
	}
}

func newPlayCmd(song *Song) *exec.Cmd {
	song.info()

	if song.Filename != "" {
		if _, err := os.Stat(song.Filename); err == nil {
			log.Printf("Playing from file: %s\n", song.Filename)
			switch Config.MediaPlayer {
			case "vlc":
				return exec.Command("vlc", "--play-and-exit", "--fullscreen", "-I", "dummy", song.Filename)
			default:
				return exec.Command("omxplayer", song.Filename)
			}
		}
	} else if song.YoutubeURL() != "" {
		log.Println("Streaming file")

		switch Config.MediaPlayer {
		case "vlc":
			return exec.Command("vlc", "--play-and-exit", "--fullscreen", "-I", "dummy", song.YoutubeURL())
		default:
			return exec.Command(Config.ScriptsFolder+"/omxplayer/start.sh", song.YoutubeURL())
		}
	}

	log.Printf("Song %d: Could not build command\n", song.ID)
	return nil
}

//============
// Song
func (song Song) YoutubeURL() string {
	return fmt.Sprintf("https://www.youtube.com/watch?v=%s", song.YoutubeID)
}

func (song Song) String() string {
	return fmt.Sprintf("%s: %s", song.Artist, song.Name)
}

func (song *Song) info() {
	s := reflect.ValueOf(song).Elem()
	typeOfT := s.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fmt.Printf("%d: %s %s = %v\n", i,
			typeOfT.Field(i).Name, f.Type(), f.Interface())
	}
}

//============
// SongHistory
func (sh *SongHistory) Add(s *Song) {
	sh.songs = append(sh.songs, s)
}

func (sh SongHistory) Songs() []*Song {
	return sh.songs
}

func (sh *SongHistory) String() string {
	return fmt.Sprintf("%v", sh.songs)
}

//============
// Karaoke
func (k Karaoke) History() SongHistory {
	return k.history
}

func (k Karaoke) Play(song *Song) {
	log.Printf("Playing %s: %s (%s)\n", song.Artist, song.Name, song.YoutubeURL())
	if cmd := newPlayCmd(song); cmd != nil {
		if output, err := cmd.Output(); err != nil {
			log.Printf("%v\n", cmd)
			log.Printf("Could not play: %v - %s\n", err, string(output))
		}
	}
	log.Printf("Finished: %s: %s\n", song.Artist, song.Name)
}

func (k *Karaoke) Run() {
	for {
		select {
		case song := <-k.Queue:
			log.Printf("%s\n", song)
			hub.broadcast <- fmt.Sprintf("[PLAYING] %s", song)
			time.Sleep(3 * time.Second)
			k.Play(song)
			k.history.Add(song)
		}
	}
}
