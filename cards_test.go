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

func TestParseLang(t *testing.T) {
	for in, want := range map[string]string{"": "EN", "es": "ES", "ES": "ES", "en": "EN", "fr": "EN"} {
		if got := parseLang(in); got != want {
			t.Errorf("parseLang(%q) = %q, want %q", in, got, want)
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
