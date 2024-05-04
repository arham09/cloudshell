package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"time"
)

//go:generate npm i
//go:embed node_modules public
var StaticFS embed.FS

func ServeAsset(w http.ResponseWriter, r *http.Request) {
	requestedFile := r.URL.Path[len("/assets/"):]
	fsys, err := fs.Sub(StaticFS, "node_modules")
	if err != nil {
		log.Fatal(err)
	}
	filesystem := http.FS(fsys)

	file, err := filesystem.Open(requestedFile)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	http.ServeContent(w, r, requestedFile, time.Time{}, file)
}
