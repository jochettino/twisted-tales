package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
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

// resolveLang picks the language for a request, in this order: the saved
// language (the lang cookie), the ?lang= value (what a shared link carries),
// the browser's Accept-Language header, and finally the default. Only the
// language switch (see /lang/{code} in main.go) overrides a saved language.
func resolveLang(saved, query, acceptLanguage string) string {
	for _, s := range []string{saved, query} {
		if s = strings.ToUpper(s); slices.Contains(languages, s) {
			return s
		}
	}
	if lang := fromAcceptLanguage(acceptLanguage); lang != "" {
		return lang
	}
	return languages[0]
}

// fromAcceptLanguage returns the supported language the browser prefers, or ""
// if the header names none. Browsers send it automatically, e.g.
// "es-ES,es;q=0.9,en;q=0.8": tags are language[-region] and q is the
// preference from 0 to 1 (1 when missing, 0 meaning "not acceptable"). The
// region is ignored, so es-MX and es-AR both count as Spanish. Ties go to the
// language listed first.
func fromAcceptLanguage(header string) string {
	if len(header) > 1024 { // far more than any real browser sends
		header = header[:1024]
	}
	best, bestQ := "", 0.0
	for _, part := range strings.Split(header, ",") {
		tag, params, _ := strings.Cut(part, ";")
		primary, _, _ := strings.Cut(strings.TrimSpace(tag), "-")
		q := 1.0
		for _, param := range strings.Split(params, ";") {
			if name, value, ok := strings.Cut(strings.TrimSpace(param), "="); ok && strings.EqualFold(strings.TrimSpace(name), "q") {
				if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v >= 0 && v <= 1 {
					q = v
				}
			}
		}
		if q <= bestQ {
			continue
		}
		for _, lang := range languages {
			if strings.EqualFold(primary, lang) {
				best, bestQ = lang, q
				break
			}
		}
	}
	return best
}

// loadCards reads every JSON file in resources/ (xxx-smalltitle-LANG.json)
// and groups the cards by language, ordered by file name so the numeric prefix
// defines the card order. Every language must have the same cards.
func loadCards() (map[string][]Card, error) {
	return loadCardsFrom(resources, "resources")
}

func loadCardsFrom(fsys fs.ReadDirFS, dir string) (map[string][]Card, error) {
	entries, err := fsys.ReadDir(dir)
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
		data, err := fs.ReadFile(fsys, dir+"/"+e.Name())
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
		return nil, fmt.Errorf("no cards found in %s/", dir)
	}
	return cards, nil
}

// cardData is the JSON served by /api/cards/{n}: just the data of a card and
// its neighbours, with no markup and no UI texts (the page already carries
// those, and they only change with a full reload when the language changes).
type cardData struct {
	Number   int    `json:"number"`
	Total    int    `json:"total"`
	Title    string `json:"title"`
	Prompt   string `json:"prompt"`
	Solution string `json:"solution,omitempty"` // only when the solution was requested
	Prev     *int   `json:"prev"`               // null when there is no previous card
	Next     *int   `json:"next"`               // null when there is no next card
}

func (p page) data() cardData {
	d := cardData{Number: p.Number, Total: p.Total, Title: p.Card.Title, Prompt: p.Card.Prompt}
	if p.Solution {
		d.Solution = p.Card.Solution
	}
	if p.Prev != 0 {
		d.Prev = &p.Prev
	}
	if p.Next != 0 {
		d.Next = &p.Next
	}
	return d
}

// newPage builds the page for card n (1-based) in the given language.
// It reports false when n is out of range.
func newPage(cards map[string][]Card, lang string, n int, solution bool) (page, bool) {
	deck := cards[lang]
	if n < 1 || n > len(deck) {
		return page{}, false
	}
	other := languages[0]
	if lang == other {
		other = languages[1]
	}
	p := page{
		Card:     deck[n-1],
		Number:   n,
		Total:    len(deck),
		Solution: solution,
		Lang:     lang,
		Other:    other,
		L:        ui[lang],
		RepoURL:  repoURL,
		CardsURL: cardsSourceURL,
		SiteURL:  siteURL,
	}
	if n > 1 {
		p.Prev = n - 1
	}
	if n < len(deck) {
		p.Next = n + 1
	}
	return p, true
}
