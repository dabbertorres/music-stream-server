package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dhowden/tag"
	"github.com/gorilla/mux"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("webapp/index.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
	}
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	var (
		artist = r.Form.Get("artist")
		album  = r.Form.Get("album")
		title  = r.Form.Get("title")
	)

	// don't allow fully empty searches
	if artist == "" && album == "" && title == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// surround the search parameters with wildcards (or make an empty search parameter a wildcard)
	artist = "%" + artist + "%"
	album = "%" + album + "%"
	title = "%" + title + "%"

	conn := dbConn(r)

	rows, err := conn.QueryContext(r.Context(), "select artist, album, title from songs where artist like ? and album like ? and title like ? order by artist, album", &artist, &album, &title)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	results := make([]Song, 0, 8)
	for rows.Next() {
		s := Song{}
		err = rows.Scan(&s.Artist, &s.Album, &s.Title)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			rows.Close()
			return
		}
		results = append(results, s)
	}

	if rows.Err() != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(rows.Err())
		return
	}

	err = json.NewEncoder(w).Encode(results)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
	}
}

func streamHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var (
		artist = vars["artist"]
		album  = vars["album"]
		title  = vars["title"]
	)

	conn := dbConn(r)

	result := Song{}
	row := conn.QueryRowContext(r.Context(), "select * from songs where artist = ? and album = ? and title = ?", artist, album, title)
	err := row.Scan(&result.Artist, &result.Album, &result.Title, &result.Path)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	http.ServeFile(w, r, result.Path)
}

func artHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	var (
		artist = vars["artist"]
		album  = vars["album"]
		title  = vars["title"]
	)

	conn := dbConn(r)

	var path string
	row := conn.QueryRowContext(r.Context(), "select path from songs where artist = ? and album = ? and title = ?", artist, album, title)
	err := row.Scan(&path)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		// no tags? skip it
		w.WriteHeader(http.StatusNoContent)
		log.Println(err)
		return
	}
	defer file.Close()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Println(err)
		return
	}

	coverArt := metadata.Picture().Data

	if coverArt == nil {
		// if we don't locally have any cover art, grab some from the Cover Art Archive
		// TODO get MusicBrainz identifier for the song (album?) and then get the corresponding (front) cover art to respond with
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.ServeContent(w, r, path, time.Time{}, bytes.NewReader(coverArt))
}
