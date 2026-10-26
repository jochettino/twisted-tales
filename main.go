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

func newMux(cards map[string][]Card) *http.ServeMux {
	mux := http.NewServeMux()
	registerStatic(mux)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		// Resume at the saved card, if it still exists. An explicit ?lang= is
		// carried to the card page, but a card number is never taken from the
		// URL: a shared link must not move someone's saved place.
		query := r.URL.Query().Get("lang")
		lang := resolveLang(query, cookieValue(r, langCookie))
		n, err := strconv.Atoi(cookieValue(r, cardCookie))
		if err != nil || n < 1 || n > len(cards[lang]) {
			n = 1
		}
		target := "/cards/" + strconv.Itoa(n)
		if slices.Contains(languages, strings.ToUpper(query)) {
			target += "?lang=" + lang
		}
		http.Redirect(w, r, target, http.StatusFound)
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

// cardForRequest resolves the card a request asks for (the {n} path value,
// ?lang= and ?solution=1) and, if it exists, remembers it in the cookies.
func cardForRequest(cards map[string][]Card, w http.ResponseWriter, r *http.Request) (page, bool) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		return page{}, false
	}
	q := r.URL.Query()
	lang := resolveLang(q.Get("lang"), cookieValue(r, langCookie))
	p, ok := newPage(cards, lang, n, q.Get("solution") == "1")
	if !ok {
		return page{}, false
	}
	setCookie(w, r, langCookie, lang)
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
