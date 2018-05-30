package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	var err error

	router := mux.NewRouter()
	router.Use(dbMiddle)

	router.Path("/search").Methods(http.MethodGet).HandlerFunc(searchHandler)
	router.Path("/stream/{artist}/{album}/{title}").Methods(http.MethodGet).HandlerFunc(streamHandler)
	router.Path("/art/{artist}/{album}/{title}").Methods(http.MethodGet).HandlerFunc(artHandler)

	server := &http.Server{
		Addr:    ":http",
		Handler: router,
	}

	err = initDb()
	if err != nil {
		log.Println(err)
		return
	}

	log.Println("Server shutdown:", server.ListenAndServe())
	closeDb()
}
