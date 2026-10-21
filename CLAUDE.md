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
- Routes: `/` redirects to `/cards/1`; `/cards/{n}` renders the card. The language is chosen with `?lang=ES`/`?lang=EN` (default EN) and carried on every link. The solution is shown only with `?solution=1`, so navigating Back/Next always lands on the prompt. The UI is plain server-rendered HTML with no JavaScript; the help section is a `<details>` element.
- Back/Next do not wrap around; they are disabled at the first and last card.

## Git

Commits are authored as `jochettino` with a GitHub noreply email (set in the global git config). Don't substitute a personal email.
