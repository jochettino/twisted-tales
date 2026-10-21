# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Twisted Tales is a lateral-thinking mystery card game (see `README.md` for the rules). This repo holds a small Go web prototype that serves one card at a time.

## Commands

- Run: `go run .` (serves on http://localhost:8080; set the `PORT` environment variable to use another port)
- Vet/format: `go vet ./...` and `gofmt -l .`
- There are no tests yet.

## Architecture

No external dependencies (stdlib only, Go 1.22+ for method/path-pattern routing). Single `main` package split by concern:

- `main.go`: HTTP server wiring and routes only.
- `cards.go`: application logic (card loading, language handling, building the page data for a card).
- `template.go`: templating (UI label strings, the `page` view model, rendering).
- `templates/card.html`: the frontend (HTML and CSS), embedded into the binary.

- Cards are JSON files in `resources/`, named `xxx-smalltitle-LANG.json` (e.g. `001-hiccups-EN.json`, `001-hiccups-ES.json`), with fields `title`, `prompt` and `solution`. They are embedded into the binary with `go:embed`, so adding or editing a card requires restarting the server.
- Languages are EN (default) and ES, defined in `languages` in `cards.go`. Every card must exist in every language; startup fails if the card sets differ. UI strings live in the `ui` map in `template.go`. The language is chosen with `?lang=ES`/`?lang=EN` and carried on every link.
- Card order is the sorted file name, so the numeric prefix defines the order. Card numbers in URLs are 1-based positions in that order, not the file prefix.
- Routes: `/` redirects to `/cards/1`; `/cards/{n}` renders the card. The solution is shown only with `?solution=1`, so navigating Back/Next always lands on the front. The UI is plain server-rendered HTML with no JavaScript.
- Back/Next do not wrap around; they are disabled at the first and last card.

## Git

Commits are authored as `jochettino` with a GitHub noreply email (set in the global git config). Don't substitute a personal email.
