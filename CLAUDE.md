# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Twisted Tales is a lateral-thinking mystery card game. The rules are in `docs/how-to-play.md`; `README.md` covers running and testing. This repo holds a small Go web prototype that serves one card at a time, in English or Spanish.

## Commands

- Run: `go run .` (serves on http://localhost:8080; set the `PORT` environment variable to use another port)
- Test: `go test ./...`; a single test: `go test -run TestLanguageSwitch ./...`
- Vet/format: `go vet ./...` and `gofmt -l .`

## Architecture

No external dependencies (stdlib only, Go 1.22+ for method/path-pattern routing). Single `main` package split by concern:

- `main.go`: server startup (`PORT`) and routes (`newMux`).
- `cards.go`: application logic (card loading, language handling, building the page data for a card).
- `template.go`: templating (UI label and help strings per language, the `page` view model, rendering).
- `templates/card.html`: the frontend (HTML and CSS), embedded into the binary.
- `*_test.go`: tests for card loading/navigation (`cards_test.go`) and HTTP routes (`main_test.go`). Route tests load the real cards in `resources/`.

Behaviour that spans several files:

- Cards are JSON files in `resources/`, named `xxx-smalltitle-LANG.json` (e.g. `001-hiccups-EN.json`, `001-hiccups-ES.json`), with fields `title`, `prompt` and `solution`. They are embedded with `go:embed`, so adding or editing a card requires restarting the server.
- Languages are EN (default) and ES, defined in `languages` in `cards.go`. Every card must exist in every language; loading fails (and the server won't start) if the card sets differ. The language switch assumes exactly two languages (`Other` in `newPage`), so adding a third also means reworking that, plus the `ui` map in `template.go`.
- Card order is the sorted file name, so the numeric prefix defines the order. Card numbers in URLs are 1-based positions in that order, not the file prefix.
- Routes: `/` redirects to a card (see user state below); `/cards/{n}` renders the card. The language is chosen with `?lang=ES`/`?lang=EN` (default EN) and carried on every link. The solution is shown only with `?solution=1`, so navigating Back/Next always lands on the prompt. The UI is server-rendered HTML; the help and About sections are `<details>` elements. A small inline script at the bottom of `templates/card.html` (progressive enhancement, no libraries) intercepts clicks on links marked `data-nav`, fetches that URL with `&partial=1` (the server then renders only the `partial` template: language link, card and nav, defined at the end of the file, with no `<head>`, CSS, script or help sections), and swaps the matching `data-swap` regions from the response, using `pushState` for Back/Next and `replaceState` for the solution toggle. The server still renders everything, so the links work without JavaScript; the language link has no `data-nav` and does a full reload. When changing the markup, keep the `data-swap`/`data-nav`/`id="nav-*"` attributes in sync with the script, and keep every swapped region inside the `lang-link`, `card-view` or `nav-buttons` template so full page and partial stay identical.
- User state lives in the browser, never on the server: viewing a card sets two cookies, `lang` and `card` (1 year, `HttpOnly`, `SameSite=Lax`, and `Secure` only when the request is https, directly or via `X-Forwarded-Proto` as on Vercel, so plain-http localhost still works; see `setCookie` in `main.go`). `/` redirects to the saved card (falling back to 1 if it is missing or out of range). Language precedence is explicit `?lang=` > `lang` cookie > default (`resolveLang` in `cards.go`). The solution toggle is deliberately not saved.
- Back/Next do not wrap around; they are disabled at the first and last card.

## Deployment (Vercel)

This application is meant to be deployed on Vercel, so every change must stay compatible with Vercel deployments. Do not make changes that could break the deployment.

How it is deployed: the GitHub repo was simply imported into a Vercel project (https://twisted-tales-game.vercel.app), and Vercel deploys continuously. Every commit pushed to `main` is deployed automatically; there is no deploy step, script or workflow to run, and nothing special to do. The repo has no `vercel.json`; the project settings live in Vercel, so don't assume you can see or change how it builds and runs.

Keep these properties, which the current design relies on:

- Stdlib-only Go with `go.mod` at the root and a single `main` package at the repo root. Don't add dependencies, restructure the package, or move `main.go` without checking that Vercel still builds it.
- The server is stateless and must not need a writable or persistent filesystem. Cards and templates are embedded with `go:embed`; user state lives in cookies. Don't add file writes, in-memory state that must survive between requests, or a database without discussing it first.
- The listen port comes from `PORT` (`main.go`). Don't hard-code it.
- It sits behind Vercel's TLS-terminating proxy, so check `X-Forwarded-Proto` for https, not just `r.TLS` (see `setCookie`).
- Anything the page needs must be served by the app itself or embedded in it. Be careful with long-running goroutines, background jobs and anything that assumes a long-lived process.

If a change might affect the build or the runtime (new dependency, Go version bump in `go.mod`, new files outside the embedded folders, changed routes), say so up front. After pushing to `main`, check that the deployment still works (the Vercel badge in the README, or the live URL).

## Git

Commits are authored as `jochettino` with a GitHub noreply email (set in the global git config). Don't substitute a personal email.
