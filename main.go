package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
)

func main() {
	cards, err := loadCards()
	if err != nil {
		log.Fatal(err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, newMux(cards)))
}

// The user's language and last card are remembered in cookies, so the server
// keeps no state of its own.
const (
	langCookie   = "lang"
	cardCookie   = "card"
	cookieMaxAge = 365 * 24 * 60 * 60
)

// isHTTPS reports whether the client reached us over https, either directly or
// through a TLS-terminating proxy such as Vercel.
func isHTTPS(r *http.Request) bool {
	return r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
}

// setCookie marks the cookie Secure only over https, so it still works on
// http://localhost during development.
func setCookie(w http.ResponseWriter, r *http.Request, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
	})
}

func cookieValue(r *http.Request, name string) string {
	if c, err := r.Cookie(name); err == nil {
		return c.Value
	}
	return ""
}

// varyLanguage tells caches that the response depends on the language
// inputs, since the same address can now be served in different languages.
func varyLanguage(w http.ResponseWriter) {
	w.Header().Add("Vary", "Cookie, Accept-Language")
}

// resumeCard is the card to open for someone arriving without a card number:
// the saved one if it still exists, else the first.
func resumeCard(cards map[string][]Card, r *http.Request, lang string) int {
	n, err := strconv.Atoi(cookieValue(r, cardCookie))
	if err != nil || n < 1 || n > len(cards[lang]) {
		return 1
	}
	return n
}

func newMux(cards map[string][]Card) *http.ServeMux {
	mux := http.NewServeMux()
	registerStatic(mux)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		// Resume at the saved card, if it still exists. A card number is never
		// taken from the URL: a shared link must not move someone's saved place.
		varyLanguage(w)
		query := r.URL.Query().Get("lang")
		lang := resolveLang(cookieValue(r, langCookie), query, r.Header.Get("Accept-Language"))
		target := "/cards/" + strconv.Itoa(resumeCard(cards, r, lang))
		// A shared link's ?lang= only matters to someone with no saved
		// language (it wins over nothing else), so carry it on for them, and for
		// link-preview crawlers, which have no cookies either.
		if !slices.Contains(languages, strings.ToUpper(cookieValue(r, langCookie))) &&
			slices.Contains(languages, strings.ToUpper(query)) {
			target += "?lang=" + lang
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
	// The language switch: save the chosen language, then go back to the saved card.
	mux.HandleFunc("GET /lang/{code}", func(w http.ResponseWriter, r *http.Request) {
		lang := strings.ToUpper(r.PathValue("code"))
		if !slices.Contains(languages, lang) {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		setCookie(w, r, langCookie, lang)
		http.Redirect(w, r, "/cards/"+strconv.Itoa(resumeCard(cards, r, lang)), http.StatusFound)
	})
	mux.HandleFunc("GET /cards/{n}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := cardForRequest(cards, w, r)
		if !ok {
			http.NotFound(w, r)
			return
		}
		if err := renderPage(w, p); err != nil {
			log.Print(err)
		}
	})
	// The same card as JSON, for the in-page script (see templates/card.html).
	mux.HandleFunc("GET /api/cards/{n}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := cardForRequest(cards, w, r)
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "card not found"})
			return
		}
		writeJSON(w, http.StatusOK, p.data())
	})
	return mux
}

// cardForRequest resolves the card a request asks for (the {n} path value and
// ?solution=1; the language comes from resolveLang) and, if it exists,
// remembers the language and card in the cookies.
func cardForRequest(cards map[string][]Card, w http.ResponseWriter, r *http.Request) (page, bool) {
	varyLanguage(w)
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		return page{}, false
	}
	q := r.URL.Query()
	lang := resolveLang(cookieValue(r, langCookie), q.Get("lang"), r.Header.Get("Accept-Language"))
	p, ok := newPage(cards, lang, n, q.Get("solution") == "1")
	if !ok {
		return page{}, false
	}
	setCookie(w, r, langCookie, lang) // sets it if missing; otherwise rewrites the same value, which renews the expiry
	setCookie(w, r, cardCookie, strconv.Itoa(n))
	return p, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Print(err)
	}
}
