package main

type Song struct {
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Title  string `json:"title"`
	Path   string `json:"path,omitempty"`
}
