package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Song struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist,omitempty"`
	Album  string `json:"album,omitempty"`
	Path   string `json:"path"`
}

var musicDir string
var static string = "/static/"
var db sql.DB
var songs []Song

func GetSongEndpoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	fmt.Printf("Requested: %q.\n", params)
	for _, song := range getSongs() {
		if song.ID == params["id"] {
			fmt.Printf("Found song: %q.\n", song)
			json.NewEncoder(w).Encode(song)
			return
		}
	}
	json.NewEncoder(w).Encode(&Song{})
}

func GetSongsEndpoint(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode(getSongs())
}

func ReadFileMeta(file string) (string, tag.Metadata) {
	var metadata tag.Metadata

	fmt.Printf("______\nReading file metadata: %q.\n", file)

	f, err := os.Open(file)
	if err != nil {
		fmt.Printf("Error loading file: %q.\n", err)
	}
	defer f.Close()

	// Read media metadata
	metadata, err = tag.ReadFrom(f)
	if err != nil {
		fmt.Printf("Error reading metadata: %q.\n", err)
	}

	// Generate md5 sum
	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	md5sum := hex.EncodeToString(h.Sum(nil))

	return md5sum, metadata
}

func MusicWalk(path string, info os.FileInfo, err error) error {
	if !info.IsDir() {
		relativePath := filepath.Clean(strings.Replace(path, musicDir, static, 1))
		relativePathNoSpaces := strings.Replace(relativePath, " ", "%20", -1)

		md5sum, metadata := ReadFileMeta(path)
		if metadata != nil {
			fmt.Printf("Song ID: %q.\n", md5sum)
			fmt.Printf("Song name: %q.\n", metadata.Title())
			addSong(Song{
				ID:     md5sum,
				Name:   metadata.Title(),
				Artist: metadata.Artist(),
				Album:  metadata.Album(),
				Path:   relativePathNoSpaces})
		} else {
			fmt.Printf("Empty metadata: %q.\n", path)
		}
	}

	return nil
}

func ScanMedia() {
	fmt.Printf("Scanning directory: '%s'.\n", musicDir)

	filepath.Walk(musicDir, MusicWalk)
}

func RunRouter() {
	fmt.Printf("Running media server. Static directory: '%s'.\n", musicDir)

	// Create HTTP router
	router := mux.NewRouter()
	router.HandleFunc("/songs", GetSongsEndpoint).Methods("GET")
	router.HandleFunc("/songs/{id}", GetSongEndpoint).Methods("GET")

	// This will serve files under http://localhost:8000/static/<filename>
	router.PathPrefix(static).Handler(http.StripPrefix(static, http.FileServer(http.Dir(musicDir))))

	log.Fatal(http.ListenAndServe(":12345", router))
}

func InitDatabase(file string) sql.DB {
	database, err := sql.Open("sqlite3", file)
	checkErr(err)
	return *database
}

func createTable() {
	sql_table := `
	CREATE TABLE IF NOT EXISTS songs (
		ID VARCHAR(64) NOT NULL PRIMARY KEY,
		Name VARCHAR(64) NOT NULL,
		Artist VARCHAR(64) NULL,
		Album VARCHAR(64) NULL,
		Path VARCHAR(64) NOT NULL);
	`

	_, err := db.Exec(sql_table)
	checkErr(err)
}

func addSong(song Song) {
	sql_additem := `
	INSERT OR REPLACE INTO songs(
		ID,
		Name,
		Artist,
		Album,
		Path
	) values(?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(sql_additem)
	checkErr(err)

	defer stmt.Close()

	_, err2 := stmt.Exec(song.ID, song.Name, song.Artist, song.Album, song.Path)
	checkErr(err2)
}

func getSongs() []Song {
	sql_readall := `
	SELECT ID, Name, Artist, Album, Path FROM songs
	`

	rows, err := db.Query(sql_readall)
	checkErr(err)
	defer rows.Close()

	var result []Song
	for rows.Next() {
		song := Song{}
		err2 := rows.Scan(&song.ID, &song.Name, &song.Artist, &song.Album, &song.Path)
		checkErr(err2)
		result = append(result, song)
	}
	return result
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var scan bool
	var database string

	flag.BoolVar(&scan, "scan", false, "scan media files")
	flag.StringVar(&musicDir, "musicDir", ".", "the directory to serve files from. Defaults to the current dir")
	flag.StringVar(&database, "database", "media.db", "path to sqlite database file")

	flag.Parse()

	// Assign initiated DB to global var
	db = InitDatabase(database)
	createTable()

	if scan {
		ScanMedia()
	} else {
		RunRouter()
	}
}
