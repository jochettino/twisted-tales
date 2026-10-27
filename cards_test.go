package main

import (
	"strings"
	"testing"
	"testing/fstest"
)

func card(title string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(`{"title":"` + title + `","prompt":"p","solution":"s"}`)}
}

func TestLoadCardsEmbedded(t *testing.T) {
	cards, err := loadCards()
	if err != nil {
		t.Fatal(err)
	}
	for _, lang := range languages {
		if len(cards[lang]) == 0 {
			t.Errorf("no %s cards", lang)
		}
		for i, c := range cards[lang] {
			if c.Title == "" || c.Prompt == "" || c.Solution == "" {
				t.Errorf("%s card %d has an empty field: %+v", lang, i+1, c)
			}
		}
	}
}

func TestLoadCardsOrderedByFileName(t *testing.T) {
	fsys := fstest.MapFS{
		"r/002-b-EN.json": card("B"),
		"r/001-a-EN.json": card("A"),
		"r/002-b-ES.json": card("B-es"),
		"r/001-a-ES.json": card("A-es"),
	}
	cards, err := loadCardsFrom(fsys, "r")
	if err != nil {
		t.Fatal(err)
	}
	if got := cards["EN"][0].Title; got != "A" {
		t.Errorf("first EN card = %q, want A", got)
	}
	if got := cards["ES"][1].Title; got != "B-es" {
		t.Errorf("second ES card = %q, want B-es", got)
	}
}

func TestLoadCardsErrors(t *testing.T) {
	tests := []struct {
		name string
		fsys fstest.MapFS
		want string
	}{
		{"language mismatch", fstest.MapFS{
			"r/001-a-EN.json": card("A"),
			"r/002-b-EN.json": card("B"),
			"r/001-a-ES.json": card("A"),
		}, "do not match"},
		{"unknown language", fstest.MapFS{"r/001-a-FR.json": card("A")}, "unknown language"},
		{"missing suffix", fstest.MapFS{"r/001.json": card("A")}, "missing language suffix"},
		{"invalid json", fstest.MapFS{"r/001-a-EN.json": {Data: []byte("{")}}, "001-a-EN.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadCardsFrom(tt.fsys, "r")
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestResolveLang(t *testing.T) {
	// Order: saved cookie, then ?lang=, then Accept-Language, then English.
	tests := []struct{ saved, query, accept, want string }{
		{"", "", "", "EN"},
		{"es", "", "", "ES"},
		{"ES", "en", "", "ES"},   // a saved language beats the query...
		{"EN", "es", "es", "EN"}, // ...and the header
		{"", "es", "", "ES"},
		{"", "ES", "en", "ES"}, // the query beats the header
		{"", "", "es-ES,es;q=0.9", "ES"},
		{"", "fr", "es", "ES"}, // unsupported query falls through to the header
		{"fr", "", "es", "ES"}, // unsupported saved language too
		{"fr", "de", "", "EN"},
		{"", "", "fr-FR,fr;q=0.9", "EN"},
	}
	for _, tt := range tests {
		if got := resolveLang(tt.saved, tt.query, tt.accept); got != tt.want {
			t.Errorf("resolveLang(%q, %q, %q) = %q, want %q", tt.saved, tt.query, tt.accept, got, tt.want)
		}
	}
}

func TestFromAcceptLanguage(t *testing.T) {
	tests := []struct{ header, want string }{
		{"", ""},
		{"es", "ES"},
		{"ES", "ES"},
		{"es-ES", "ES"},
		{"es-MX,es;q=0.9", "ES"},
		{"en-US,en;q=0.9", "EN"},
		{"es-ES,es;q=0.9,en;q=0.8", "ES"},
		{"en-GB,en;q=0.9,es;q=0.8", "EN"},
		{"fr-FR,fr;q=0.9,es;q=0.8", "ES"}, // the first supported one, even if not first
		{"fr-FR,fr;q=0.9,de;q=0.5", ""},   // nothing supported
		{"*", ""},                         // a wildcard names no language
		{"es;q=0.5,en;q=0.9", "EN"},       // quality decides, not order
		{"en;q=0.8,es;q=0.8", "EN"},       // ties go to the first one listed
		{"es;q=0,en;q=0.1", "EN"},         // q=0 means "not acceptable"
		{"es;q=0", ""},
		{" es ; q=0.7 , en ; q=0.2 ", "ES"}, // spaces are tolerated
		{"es;Q=0.3,en;q=0.2", "ES"},         // the q name is case-insensitive
		{"es;q=abc,en;q=0.5", "ES"},         // an unreadable q counts as 1
		{"es;q=7,en;q=0.5", "ES"},           // so does an out-of-range one
		{",,;;,es", "ES"},                   // junk around a valid tag
		{"zh-Hant-TW,zh;q=0.9", ""},
		{"english", ""}, // not a language tag
		{"e", ""},
		{strings.Repeat("fr,", 600) + "es", ""}, // beyond the length cap the rest is ignored
	}
	for _, tt := range tests {
		if got := fromAcceptLanguage(tt.header); got != tt.want {
			t.Errorf("fromAcceptLanguage(%.60q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestNewPageNavigation(t *testing.T) {
	cards, err := loadCards()
	if err != nil {
		t.Fatal(err)
	}
	total := len(cards["EN"])

	first, ok := newPage(cards, "EN", 1, false)
	if !ok || first.Prev != 0 || first.Next != 2 || first.Total != total || first.Other != "ES" {
		t.Errorf("first page = %+v, ok=%v", first, ok)
	}
	last, ok := newPage(cards, "ES", total, true)
	if !ok || last.Prev != total-1 || last.Next != 0 || !last.Solution || last.Other != "EN" {
		t.Errorf("last page = %+v, ok=%v", last, ok)
	}
	for _, n := range []int{0, -1, total + 1} {
		if _, ok := newPage(cards, "EN", n, false); ok {
			t.Errorf("newPage(%d) should be out of range", n)
		}
	}
}
