package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
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
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		// Resume at the saved card, if it still exists.
		lang := resolveLang("", cookieValue(r, langCookie))
		n, err := strconv.Atoi(cookieValue(r, cardCookie))
		if err != nil || n < 1 || n > len(cards[lang]) {
			n = 1
		}
		http.Redirect(w, r, "/cards/"+strconv.Itoa(n), http.StatusFound)
	})
	mux.HandleFunc("GET /cards/{n}", func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(r.PathValue("n"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		lang := resolveLang(q.Get("lang"), cookieValue(r, langCookie))
		p, ok := newPage(cards, lang, n, q.Get("solution") == "1")
		if !ok {
			http.NotFound(w, r)
			return
		}
		setCookie(w, r, langCookie, lang)
		setCookie(w, r, cardCookie, strconv.Itoa(n))
		if err := renderPage(w, p); err != nil {
			log.Print(err)
		}
	})
	return mux
}
