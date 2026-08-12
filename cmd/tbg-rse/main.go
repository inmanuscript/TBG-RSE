package main

import (
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/inmanuscript/TBG-RSE/internal/room"
	"github.com/inmanuscript/TBG-RSE/internal/store"
	"github.com/inmanuscript/TBG-RSE/webui"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	dbPath := flag.String("db", "tbg-rse.db", "SQLite database path")
	flag.Parse()

	logger := log.New(os.Stdout, "[tbg-rse] ", log.LstdFlags|log.Lmsgprefix)

	st, err := store.Open(*dbPath)
	if err != nil {
		logger.Fatalf("open db: %v", err)
	}
	defer st.Close()

	mgr := room.NewManager(st, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", mgr.HandleWS)

	static, err := fs.Sub(webui.Assets, "dist")
	if err != nil {
		logger.Fatalf("embed dist: %v", err)
	}
	fileServer := http.FileServer(http.FS(static))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" {
			trimmed := path
			if len(trimmed) > 0 && trimmed[0] == '/' {
				trimmed = trimmed[1:]
			}
			if _, err := fs.Stat(static, trimmed); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
			// SPA fallback
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	absDB, _ := filepath.Abs(*dbPath)
	logger.Printf("listening on %s (db=%s)", *addr, absDB)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		logger.Fatal(err)
	}
}
