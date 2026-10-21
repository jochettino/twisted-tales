package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
)

//go:embed resources/*.json
var resources embed.FS

type Card struct {
	Title    string `json:"title"`
	Prompt   string `json:"prompt"`
	Solution string `json:"solution"`
}

// languages lists the supported languages; the first is the default.
// Card files carry the language as a file name suffix, e.g. 001-hiccups-EN.json.
var languages = []string{"EN", "ES"}

type labels struct {
	Card, Of, Back, Next, Show, Hide, Solution, SwitchTo string
}

var ui = map[string]labels{
	"EN": {"Card", "of", "Back", "Next", "Show solution", "Hide solution", "Solution:", "Español"},
	"ES": {"Carta", "de", "Atrás", "Siguiente", "Mostrar solución", "Ocultar solución", "Solución:", "English"},
}

type page struct {
	Card     Card
	Number   int
	Total    int
	Solution bool
	Lang     string
	Other    string // the language the switch button leads to
	L        labels
	Prev     int // 0 when there is no previous card
	Next     int // 0 when there is no next card
}

var pageTmpl = template.Must(template.New("card").Parse(`<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Twisted Tales</title>
<style>
  body { font-family: sans-serif; max-width: 40em; margin: 2em auto; padding: 0 1em; }
  .card { border: 1px solid #333; border-radius: 8px; padding: 1em 1.5em; margin: 1em 0; }
  .solution { background: #f3f3f3; }
  nav a, nav span { margin-right: 1em; }
  nav .disabled { color: #aaa; }
</style>
</head>
<body>
<h1>Twisted Tales</h1>
<p>{{.L.Card}} {{.Number}} {{.L.Of}} {{.Total}} &middot; <a href="/cards/{{.Number}}?lang={{.Other}}">{{.L.SwitchTo}}</a></p>
<div class="card">
  <h2>{{.Card.Title}}</h2>
  <p>{{.Card.Prompt}}</p>
  {{if .Solution}}
  <div class="solution"><strong>{{.L.Solution}}</strong> {{.Card.Solution}}</div>
  {{end}}
</div>
<nav>
  {{if .Prev}}<a href="/cards/{{.Prev}}?lang={{.Lang}}">&larr; {{.L.Back}}</a>{{else}}<span class="disabled">&larr; {{.L.Back}}</span>{{end}}
  {{if .Solution}}<a href="/cards/{{.Number}}?lang={{.Lang}}">{{.L.Hide}}</a>{{else}}<a href="/cards/{{.Number}}?lang={{.Lang}}&solution=1">{{.L.Show}}</a>{{end}}
  {{if .Next}}<a href="/cards/{{.Next}}?lang={{.Lang}}">{{.L.Next}} &rarr;</a>{{else}}<span class="disabled">{{.L.Next}} &rarr;</span>{{end}}
</nav>
</body>
</html>
`))

// loadCards reads every JSON file in resources/ (xxx-smalltitle-LANG.json)
// and groups the cards by language, ordered by file name so the numeric prefix
// defines the card order. Every language must have the same cards.
func loadCards() (map[string][]Card, error) {
	entries, err := resources.ReadDir("resources")
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	cards := map[string][]Card{}
	ids := map[string][]string{} // card ids (file name without language suffix) per language
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		i := strings.LastIndex(name, "-")
		if i < 0 {
			return nil, fmt.Errorf("%s: missing language suffix", e.Name())
		}
		id, lang := name[:i], name[i+1:]
		if !slices.Contains(languages, lang) {
			return nil, fmt.Errorf("%s: unknown language %q", e.Name(), lang)
		}
		data, err := resources.ReadFile("resources/" + e.Name())
		if err != nil {
			return nil, err
		}
		var c Card
		if err := json.Unmarshal(data, &c); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		cards[lang] = append(cards[lang], c)
		ids[lang] = append(ids[lang], id)
	}

	for _, lang := range languages[1:] {
		if !slices.Equal(ids[languages[0]], ids[lang]) {
			return nil, fmt.Errorf("%s cards %v do not match %s cards %v", lang, ids[lang], languages[0], ids[languages[0]])
		}
	}
	if len(cards[languages[0]]) == 0 {
		return nil, fmt.Errorf("no cards found in resources/")
	}
	return cards, nil
}

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
		lang := strings.ToUpper(r.URL.Query().Get("lang"))
		if !slices.Contains(languages, lang) {
			lang = languages[0]
		}
		other := languages[0]
		if lang == other {
			other = languages[1]
		}
		deck := cards[lang]
		n, err := strconv.Atoi(r.PathValue("n"))
		if err != nil || n < 1 || n > len(deck) {
			http.NotFound(w, r)
			return
		}
		p := page{
			Card:     deck[n-1],
			Number:   n,
			Total:    len(deck),
			Lang:     lang,
			Other:    other,
			L:        ui[lang],
			Solution: r.URL.Query().Get("solution") == "1",
		}
		if n > 1 {
			p.Prev = n - 1
		}
		if n < len(deck) {
			p.Next = n + 1
		}
		if err := pageTmpl.Execute(w, p); err != nil {
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
