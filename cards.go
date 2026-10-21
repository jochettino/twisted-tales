package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
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

// parseLang returns a supported language, falling back to the default.
func parseLang(s string) string {
	s = strings.ToUpper(s)
	if slices.Contains(languages, s) {
		return s
	}
	return languages[0]
}

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
	}
	if n > 1 {
		p.Prev = n - 1
	}
	if n < len(deck) {
		p.Next = n + 1
	}
	return p, true
}
