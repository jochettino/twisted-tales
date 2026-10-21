package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	cards, err := loadCards()
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/cards/1", http.StatusFound)
	})
	mux.HandleFunc("GET /cards/{n}", func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(r.PathValue("n"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		p, ok := newPage(cards, parseLang(q.Get("lang")), n, q.Get("solution") == "1")
		if !ok {
			http.NotFound(w, r)
			return
		}
		if err := renderPage(w, p); err != nil {
			log.Print(err)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
