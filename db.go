package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dhowden/tag"
)

const (
	dbConnKey = "dbconnkey"
)

var (
	db *sql.DB
)

func dbMiddle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := db.Conn(r.Context())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
		// just one line, but a defer ensures it gets called if the handler panics for some reason
		defer conn.Close()
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), dbConnKey, conn)))
	})
}

func dbConn(r *http.Request) (*sql.Conn, error) {
	return r.Context().Value(dbConnKey).(*sql.DB).Conn(r.Context())
}

func initDb() (err error) {
	db, err = sql.Open("sqlite3", "./songs.db")
	if err != nil {
		return
	}

	_, err = db.Exec(`create table if not exists songs (
	artist varchar(32) not null,
	album varchar(32) not null,
	title varchar(64) not null,
	path varchar(64) not null,
	primary key (artist, album, title))`)

	if err != nil {
		return
	}

	tx, err := db.Begin()
	if err != nil {
		return
	}

	addStmt, err := tx.Prepare("insert into songs values (?, ?, ?, ?)")
	if err != nil {
		return
	}

	songsDir, err := filepath.EvalSymlinks("./songs/")
	if err != nil {
		return
	}

	err = filepath.Walk(songsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		metadata, err := tag.ReadFrom(file)
		if err != nil {
			// no tags? skip it
			return nil
		}

		var (
			title  = metadata.Title()
			album  = metadata.Album()
			artist = metadata.Artist()
		)

		path = filepath.ToSlash(path)
		_, err = addStmt.Exec(&artist, &album, &title, &path)
		return err
	})

	if err != nil {
		tx.Rollback()
	} else {
		err = tx.Commit()
	}

	return
}

func closeDb() {
	db.Close()
}
