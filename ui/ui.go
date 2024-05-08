package ui

import (
	"embed"
	"fmt"
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

func ServePublic() (fs.FS, error) {
	public, err := fs.Sub(StaticFS, "public")
	if err != nil {
		return nil, fmt.Errorf("failed to get public embedded dir: %s", err)

	}

	return public, nil
}
