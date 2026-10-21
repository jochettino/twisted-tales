package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func get(t *testing.T, path string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return serve(t, req)
}

func serve(t *testing.T, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	cards, err := loadCards()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	newMux(cards).ServeHTTP(rec, req)
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

func cookie(name, value string) *http.Cookie { return &http.Cookie{Name: name, Value: value} }

func setCookies(rec *httptest.ResponseRecorder) map[string]string {
	got := map[string]string{}
	for _, c := range rec.Result().Cookies() {
		got[c.Name] = c.Value
	}
	return got
}

func TestViewingACardSavesLanguageAndCard(t *testing.T) {
	got := setCookies(get(t, "/cards/3?lang=es"))
	if got["lang"] != "ES" || got["card"] != "3" {
		t.Errorf("cookies = %v, want lang=ES card=3", got)
	}
	for _, c := range get(t, "/cards/3").Result().Cookies() {
		if !c.HttpOnly || c.MaxAge <= 0 || c.Path != "/" || c.SameSite != http.SameSiteLaxMode {
			t.Errorf("cookie %s has unexpected attributes: %+v", c.Name, c)
		}
	}
}

func TestNotFoundDoesNotSaveState(t *testing.T) {
	if got := setCookies(get(t, "/cards/99?lang=es")); len(got) != 0 {
		t.Errorf("404 set cookies: %v", got)
	}
}

func TestHomeResumesAtSavedCard(t *testing.T) {
	rec := get(t, "/", cookie("card", "3"), cookie("lang", "ES"))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/cards/3" {
		t.Errorf("/ = %d -> %q, want 302 -> /cards/3", rec.Code, rec.Header().Get("Location"))
	}
}

func TestHomeIgnoresBadSavedCard(t *testing.T) {
	for _, v := range []string{"99", "0", "-2", "abc", ""} {
		rec := get(t, "/", cookie("card", v))
		if got := rec.Header().Get("Location"); got != "/cards/1" {
			t.Errorf("card cookie %q: / -> %q, want /cards/1", v, got)
		}
	}
}

func TestSavedLanguageIsUsedWithoutQuery(t *testing.T) {
	if body := get(t, "/cards/2", cookie("lang", "ES")).Body.String(); !strings.Contains(body, "Carta 2 de") {
		t.Error("saved ES language was not used")
	}
}

func TestExplicitLanguageBeatsSavedLanguage(t *testing.T) {
	rec := get(t, "/cards/2?lang=en", cookie("lang", "ES"))
	if !strings.Contains(rec.Body.String(), "Card 2 of") {
		t.Error("?lang=en should override the saved ES language")
	}
	if got := setCookies(rec); got["lang"] != "EN" {
		t.Errorf("lang cookie = %q, want EN", got["lang"])
	}
}

func TestInvalidSavedLanguageFallsBackToDefault(t *testing.T) {
	if body := get(t, "/cards/2", cookie("lang", "fr")).Body.String(); !strings.Contains(body, "Card 2 of") {
		t.Error("invalid saved language should fall back to EN")
	}
}

func TestCookiesSecureOnlyOverHTTPS(t *testing.T) {
	plain := httptest.NewRequest("GET", "/cards/1", nil)
	forwarded := httptest.NewRequest("GET", "/cards/1", nil)
	forwarded.Header.Set("X-Forwarded-Proto", "https")
	direct := httptest.NewRequest("GET", "https://example.com/cards/1", nil)
	direct.TLS = &tls.ConnectionState{}

	tests := []struct {
		name string
		req  *http.Request
		want bool
	}{
		{"plain http (localhost)", plain, false},
		{"behind an https proxy", forwarded, true},
		{"direct TLS", direct, true},
	}
	for _, tt := range tests {
		cookies := serve(t, tt.req).Result().Cookies()
		if len(cookies) != 2 {
			t.Fatalf("%s: got %d cookies, want 2", tt.name, len(cookies))
		}
		for _, c := range cookies {
			if c.Secure != tt.want {
				t.Errorf("%s: cookie %s Secure = %v, want %v", tt.name, c.Name, c.Secure, tt.want)
			}
		}
	}
}
