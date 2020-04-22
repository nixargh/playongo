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
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"github.com/dhowden/tag"
	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

var version string = "0.5.0"

var musicDir string
var static string = "/static/"
var db sql.DB
var songs []song

type song struct {
	ID       string       `json:"id,omitempty"`
	Name     string       `json:"name,omitempty"`
	Artist   string       `json:"artist,omitempty"`
	Album    string       `json:"album,omitempty"`
	Genre    string       `json:"genre,omitempty"`
	Year     int          `json:"year,omitempty"`
	Format   tag.Format   `json:"format,omitempty"`
	FileType tag.FileType `json:"filetype,omitempty"`
	Path     string       `json:"path,omitempty"`
	Size     int64        `json:"size,omitempty"`
}

// Router part

func runRouter() {
	fmt.Printf("Running media server. Static directory: '%s'.\n", musicDir)

	// Create HTTP router
	router := mux.NewRouter()
	router.HandleFunc("/songs", getSongEndpoint).Methods("GET")
	router.HandleFunc("/songs/{id}", getSongByID).Methods("GET")
	router.HandleFunc("/songs/{attribute}/{value}", getSongByAttribute).Methods("GET")

	// This will serve files under http://localhost:8000/static/<filename>
	router.PathPrefix(static).Handler(http.StripPrefix(static, http.FileServer(http.Dir(musicDir))))

	log.Fatal(http.ListenAndServe(":12345", router))
}

func findRealAttribute(attribute string) string {
	fmt.Printf("Looking for a real name of attribute: %q.\n", attribute)
	var realAttribute string

	var possibleAttrs [3]string
	possibleAttrs[0] = attribute
	possibleAttrs[1] = strings.ToUpper(attribute)
	possibleAttrs[2] = strings.Title(attribute)

	s := song{}
	sv := reflect.TypeOf(s)
	for _, fieldName := range possibleAttrs {
		//fmt.Printf("Checking attribute: %q.\n", fieldName)
		field, found := sv.FieldByName(fieldName)
		if !found {
			continue
		}
		realAttribute = field.Name
		break
	}

	fmt.Printf("Real attribute: %q.\n", realAttribute)
	return realAttribute
}

func getSongs(attribute string, value string) []song {
	songs := make([]song, 0)

	fmt.Printf("Looking for song(s) with attribute %q = %q.\n", attribute, value)

	var rows *sql.Rows
	var err error

	if attribute == "" {
		fmt.Printf("Attribute is empty so returning all songs.\n")
		rows, err = db.Query("SELECT ID, Name, Artist, Album, Genre, Year, Format, FileType, Path, Size FROM songs")
	} else {
		query := fmt.Sprintf("SELECT ID, Name, Artist, Album, Genre, Year, Format, FileType, Path, Size FROM songs where %s = ?", attribute)
		rows, err = db.Query(query, value)
	}

	checkErr(err)
	defer rows.Close()

	for rows.Next() {
		song := song{}
		err2 := rows.Scan(
			&song.ID,
			&song.Name,
			&song.Artist,
			&song.Album,
			&song.Genre,
			&song.Year,
			&song.Format,
			&song.FileType,
			&song.Path,
			&song.Size)
		checkErr(err2)
		songs = append(songs, song)
	}

	fmt.Printf("Found number of songs: %d.", len(songs))
	for _, song := range songs {
		fmt.Printf("Found song: %q.\n", song)
	}
	return songs
}

func getSongByID(w http.ResponseWriter, req *http.Request) {
	var songs []song

	params := mux.Vars(req)
	id := params["id"]
	fmt.Printf("Requested ID: %q.\n", id)
	songs = getSongs("ID", id)
	json.NewEncoder(w).Encode(&songs)
}

func getSongByAttribute(w http.ResponseWriter, req *http.Request) {
	var songs []song
	params := mux.Vars(req)
	fmt.Printf("Requested attribute: %q.\n", params)

	realAttribute := findRealAttribute(params["attribute"])
	if realAttribute != "" {
		songs = getSongs(realAttribute, params["value"])
		json.NewEncoder(w).Encode(&songs)
	} else {
		err := "Bad request: Attribute hasn't been found.\n"
		fmt.Printf(err)
		http.Error(w, err, http.StatusBadRequest)
	}
}

func getSongEndpoint(w http.ResponseWriter, req *http.Request) {
	json.NewEncoder(w).Encode(getSongs("", ""))
}

// Scan part

func scanMedia() {
	fmt.Printf("Scanning directory: '%s'.\n", musicDir)

	songsList := make(chan string, 100)
	go mediaWalk(musicDir, songsList)

	var wg sync.WaitGroup
	var mux sync.Mutex

	for songFile := range songsList {
		wg.Add(1)
		fmt.Printf("Song file: %q.\n", songFile)
		go readMeta(songFile, &wg, &mux)
	}
	wg.Wait()

	fmt.Printf("Scan has been finished.\n")
}

func mediaWalk(musicDir string, songsList chan string) {
	filepath.Walk(musicDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			songsList <- path
		}
		return nil
	})

	close(songsList)
}

func readMeta(path string, wg *sync.WaitGroup, mux *sync.Mutex) {
	defer wg.Done()

	relativePath := filepath.Clean(strings.Replace(path, musicDir, static, 1))
	//relativePathNoSpaces := strings.Replace(relativePath, " ", "%20", -1)
	relativePathNoSpaces := &url.URL{Path: relativePath}

	md5sum, metadata, file_size := readFileMetadata(path)
	if metadata != nil {
		fmt.Printf("song ID: %q.\n", md5sum)
		fmt.Printf("song name: %q.\n", metadata.Title())

		mux.Lock()
		addSong(song{
			ID:       md5sum,
			Name:     metadata.Title(),
			Artist:   metadata.Artist(),
			Album:    metadata.Album(),
			Genre:    metadata.Genre(),
			Year:     metadata.Year(),
			Format:   metadata.Format(),
			FileType: metadata.FileType(),
			Path:     relativePathNoSpaces.String(),
			Size:     file_size})
		mux.Unlock()
	} else {
		fmt.Printf("Empty metadata: %q.\n", path)
	}
}

func readFileMetadata(file string) (string, tag.Metadata, int64) {
	var metadata tag.Metadata

	fmt.Printf("Reading file metadata: %q. ", file)

	f, err := os.Open(file)
	file_stat, err := f.Stat()
	file_size := file_stat.Size()
	fmt.Printf(" Size: %v ", file_size)
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
	fmt.Printf("\n")

	return md5sum, metadata, file_size
}

func addSong(song song) {
	query := `
	INSERT OR REPLACE INTO songs(
		ID,
		Name,
		Artist,
		Album,
		Genre,
		Year,
		Format,
		FileType,
		Path,
		Size
	) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := db.Prepare(query)
	checkErr(err)

	defer stmt.Close()

	_, err2 := stmt.Exec(
		song.ID,
		song.Name,
		song.Artist,
		song.Album,
		song.Genre,
		song.Year,
		song.Format,
		song.FileType,
		song.Path,
		song.Size)
	checkErr(err2)
}

// Database part

func initDatabase(file string) sql.DB {
	database, err := sql.Open("sqlite3", file)
	checkErr(err)
	return *database
}

func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS songs (
		ID VARCHAR(64) NOT NULL PRIMARY KEY,
		Name VARCHAR(64) NOT NULL,
		Artist VARCHAR(64) NULL,
		Album VARCHAR(64) NULL,
		Genre VARCHAR(64) NULL,
		Year VARCHAR(64) NULL,
		Format VARCHAR(64) NULL,
		FileType VARCHAR(64) NULL,
		Path VARCHAR(64) NOT NULL,
		Size INTEGER NOT NULL);
	`
	_, err := db.Exec(query)
	checkErr(err)
}

// Other

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
	db = initDatabase(database)
	createTable()

	if scan {
		scanMedia()
	} else {
		runRouter()
	}
}
