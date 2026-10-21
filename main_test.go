package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	cards, err := loadCards()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	newMux(cards).ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
	return rec
}

func TestRoutes(t *testing.T) {
	if rec := get(t, "/"); rec.Code != http.StatusFound || rec.Header().Get("Location") != "/cards/1" {
		t.Errorf("/ = %d -> %q, want 302 -> /cards/1", rec.Code, rec.Header().Get("Location"))
	}
	for _, path := range []string{"/cards/0", "/cards/99", "/cards/x", "/nope"} {
		if rec := get(t, path); rec.Code != http.StatusNotFound {
			t.Errorf("%s = %d, want 404", path, rec.Code)
		}
	}
}

func TestSolutionHiddenByDefault(t *testing.T) {
	cards, _ := loadCards()
	solution := cards["EN"][0].Solution

	if body := get(t, "/cards/1").Body.String(); strings.Contains(body, "Solution:") || strings.Contains(body, solution) {
		t.Error("solution is visible without ?solution=1")
	}
	if body := get(t, "/cards/1?solution=1").Body.String(); !strings.Contains(body, "Solution:") {
		t.Error("solution is not visible with ?solution=1")
	}
}

func TestLanguageSwitch(t *testing.T) {
	en := get(t, "/cards/2").Body.String()
	if !strings.Contains(en, "Card 2 of") || !strings.Contains(en, `href="/cards/2?lang=ES"`) {
		t.Error("English page should say 'Card 2 of' and link to the Spanish version of card 2")
	}
	es := get(t, "/cards/2?lang=es").Body.String()
	if !strings.Contains(es, "Carta 2 de") || !strings.Contains(es, `href="/cards/2?lang=EN"`) {
		t.Error("Spanish page should say 'Carta 2 de' and link to the English version of card 2")
	}
	if !strings.Contains(es, `href="/cards/3?lang=ES"`) {
		t.Error("Next link should keep the selected language")
	}
}

func TestHelpSection(t *testing.T) {
	if body := get(t, "/cards/1").Body.String(); !strings.Contains(body, "How to play") {
		t.Error("English help section missing")
	}
	if body := get(t, "/cards/1?lang=ES").Body.String(); !strings.Contains(body, "Cómo jugar") {
		t.Error("Spanish help section missing")
	}
}
