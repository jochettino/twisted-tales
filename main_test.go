package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"html"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	solution := html.EscapeString(cards["EN"][0].Solution)

	body := get(t, "/cards/1").Body.String()
	if strings.Contains(body, solution) {
		t.Error("solution text is in the page without ?solution=1")
	}
	if !strings.Contains(body, `id="card-solution" hidden`) {
		t.Error("solution block should be hidden without ?solution=1")
	}
	body = get(t, "/cards/1?solution=1").Body.String()
	if !strings.Contains(body, solution) {
		t.Error("solution is not in the page with ?solution=1")
	}
	if strings.Contains(body, `id="card-solution" hidden`) {
		t.Error("solution block should be visible with ?solution=1")
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

func TestAboutSection(t *testing.T) {
	for path, want := range map[string]string{"/cards/1": "<summary>About</summary>", "/cards/1?lang=ES": "<summary>Acerca de</summary>"} {
		body := get(t, path).Body.String()
		if !strings.Contains(body, want) {
			t.Errorf("%s: About section missing", path)
		}
		if !strings.Contains(body, `href="`+cardsSourceURL+`"`) {
			t.Errorf("%s: cards source link missing", path)
		}
		if !strings.Contains(body, `href="`+repoURL+`"`) {
			t.Errorf("%s: repo link missing", path)
		}
	}
}

func TestPageHooksForScript(t *testing.T) {
	body := get(t, "/cards/2").Body.String()
	for _, want := range []string{
		`data-lang="EN" data-other="ES"`, `data-card="Card" data-of="of"`, `data-show="Show solution" data-hide="Hide solution"`,
		`id="lang-link"`, `id="card-counter"`, `id="card-title"`, `id="card-prompt"`, `id="card-solution-text"`,
		`id="nav-solution" data-nav="solution" href="/cards/2?lang=EN&solution=1"`,
		`id="nav-prev" data-nav="card" href="/cards/1?lang=EN"`,
		`id="nav-next" data-nav="card" href="/cards/3?lang=EN"`,
		"fetch('/api' + u.pathname + u.search", // the script survived templating
	} {
		if !strings.Contains(body, want) {
			t.Errorf("page is missing %q", want)
		}
	}
	// At the ends Back/Next stay in the page (the script re-enables them) but have no link.
	if first := get(t, "/cards/1").Body.String(); !strings.Contains(first, `id="nav-prev" data-nav="card" class="disabled"`) {
		t.Error("Back should be disabled, without href, on the first card")
	}
}

type apiCard struct {
	Number   int    `json:"number"`
	Total    int    `json:"total"`
	Title    string `json:"title"`
	Prompt   string `json:"prompt"`
	Solution string `json:"solution"`
	Prev     *int   `json:"prev"`
	Next     *int   `json:"next"`
}

func getAPI(t *testing.T, path string) (apiCard, *httptest.ResponseRecorder) {
	t.Helper()
	rec := get(t, path)
	var c apiCard
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
			t.Fatalf("%s: %v\n%s", path, err, rec.Body)
		}
	}
	return c, rec
}

func TestAPICard(t *testing.T) {
	cards, _ := loadCards()
	c, rec := getAPI(t, "/api/cards/2")
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	want := cards["EN"][1]
	if c.Number != 2 || c.Total != len(cards["EN"]) || c.Title != want.Title || c.Prompt != want.Prompt {
		t.Errorf("card = %+v, want number 2 of %d, %q", c, len(cards["EN"]), want.Title)
	}
	if c.Prev == nil || *c.Prev != 1 || c.Next == nil || *c.Next != 3 {
		t.Errorf("prev/next = %v/%v, want 1/3", c.Prev, c.Next)
	}
	// No markup in the response.
	if body := rec.Body.String(); strings.Contains(body, "<") && !strings.Contains(body, `\u003c`) {
		t.Errorf("response contains HTML: %s", body)
	}
}

func TestAPISolutionOnlyWhenAsked(t *testing.T) {
	cards, _ := loadCards()
	if c, rec := getAPI(t, "/api/cards/1"); c.Solution != "" || strings.Contains(rec.Body.String(), `"solution"`) {
		t.Errorf("solution sent without solution=1: %s", rec.Body)
	}
	if c, _ := getAPI(t, "/api/cards/1?solution=1"); c.Solution != cards["EN"][0].Solution {
		t.Errorf("solution = %q, want %q", c.Solution, cards["EN"][0].Solution)
	}
}

func TestAPILanguageAndEnds(t *testing.T) {
	cards, _ := loadCards()
	if c, _ := getAPI(t, "/api/cards/2?lang=ES"); c.Title != cards["ES"][1].Title {
		t.Errorf("ES title = %q, want %q", c.Title, cards["ES"][1].Title)
	}
	first, rec := getAPI(t, "/api/cards/1")
	if first.Prev != nil || !strings.Contains(rec.Body.String(), `"prev":null`) {
		t.Errorf("first card prev should be null: %s", rec.Body)
	}
	last, rec := getAPI(t, "/api/cards/"+strconv.Itoa(len(cards["EN"])))
	if last.Next != nil || !strings.Contains(rec.Body.String(), `"next":null`) {
		t.Errorf("last card next should be null: %s", rec.Body)
	}
}

func TestAPINotFoundIsJSON(t *testing.T) {
	for _, path := range []string{"/api/cards/0", "/api/cards/99", "/api/cards/x"} {
		rec := get(t, path)
		if rec.Code != http.StatusNotFound || !strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json") {
			t.Errorf("%s = %d %q, want a JSON 404", path, rec.Code, rec.Header().Get("Content-Type"))
		}
		if len(rec.Result().Cookies()) != 0 {
			t.Errorf("%s should not set cookies", path)
		}
	}
}

func TestAPISavesLanguageAndCard(t *testing.T) {
	if got := setCookies(get(t, "/api/cards/3?lang=ES")); got["lang"] != "ES" || got["card"] != "3" {
		t.Errorf("API cookies = %v, want lang=ES card=3", got)
	}
}

func TestStaticIcons(t *testing.T) {
	sizes := map[string]int{
		"/favicon-32x32.png": 32, "/apple-touch-icon.png": 180,
		"/android-chrome-192x192.png": 192, "/android-chrome-512x512.png": 512,
	}
	for path, want := range sizes {
		rec := get(t, path)
		if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "image/png" {
			t.Errorf("%s = %d %q, want 200 image/png", path, rec.Code, rec.Header().Get("Content-Type"))
			continue
		}
		img, err := png.Decode(bytes.NewReader(rec.Body.Bytes()))
		if err != nil || img.Bounds().Dx() != want || img.Bounds().Dy() != want {
			t.Errorf("%s: want a %dx%d PNG (err %v)", path, want, want, err)
		}
		if rec.Header().Get("Cache-Control") == "" {
			t.Errorf("%s has no Cache-Control", path)
		}
	}
	ico := get(t, "/favicon.ico")
	// An ICO starts with reserved=0, type=1 (icon).
	if ico.Code != http.StatusOK || !bytes.HasPrefix(ico.Body.Bytes(), []byte{0, 0, 1, 0}) {
		t.Errorf("/favicon.ico = %d, want a 200 ICO file", ico.Code)
	}
}

func TestWebManifest(t *testing.T) {
	rec := get(t, "/site.webmanifest")
	if rec.Code != http.StatusOK || rec.Header().Get("Content-Type") != "application/manifest+json" {
		t.Fatalf("manifest = %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	var m struct {
		ShortName  string `json:"short_name"`
		StartURL   string `json:"start_url"`
		Theme      string `json:"theme_color"`
		Background string `json:"background_color"`
		Icons      []struct {
			Src, Sizes, Purpose string
		} `json:"icons"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m.ShortName != "Twisted Tales" || m.StartURL != "/" || m.Theme != "#000000" || m.Background != "#000000" {
		t.Errorf("manifest = %+v", m)
	}
	if len(m.Icons) == 0 {
		t.Fatal("manifest has no icons")
	}
	for _, ic := range m.Icons {
		if ic.Purpose == "maskable" {
			t.Errorf("%s is marked maskable, but the moon is not padded for a mask", ic.Src)
		}
		if get(t, ic.Src).Code != http.StatusOK {
			t.Errorf("manifest icon %s is not served", ic.Src)
		}
	}
}

func TestPageHeadLinksIconsAndPreview(t *testing.T) {
	for path, tagline := range map[string]string{"/cards/1": "lateral-thinking", "/cards/1?lang=ES": "pensamiento lateral"} {
		body := get(t, path).Body.String()
		for _, want := range []string{
			`<link rel="icon" href="/favicon.ico" sizes="any">`,
			`<link rel="icon" type="image/png" sizes="32x32" href="/favicon-32x32.png">`,
			`<link rel="apple-touch-icon" href="/apple-touch-icon.png">`,
			`<link rel="manifest" href="/site.webmanifest">`,
			`<meta property="og:image" content="` + siteURL + `/android-chrome-512x512.png">`,
			tagline,
		} {
			if !strings.Contains(body, want) {
				t.Errorf("%s: head is missing %q", path, want)
			}
		}
	}
	// A link preview must not leak anything about the card.
	cards, _ := loadCards()
	if body := get(t, "/cards/1").Body.String(); strings.Contains(body, `og:title" content="`+html.EscapeString(cards["EN"][0].Title)) {
		t.Error("og:title should be the site name, not the card title")
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
