# Twisted Tales

[![Vercel status](https://img.shields.io/website?url=https%3A%2F%2Ftwisted-tales-eight.vercel.app&up_message=OK&down_message=KO&up_color=brightgreen&down_color=red&label=Vercel)](https://twisted-tales-eight.vercel.app) [![Build status](https://img.shields.io/github/actions/workflow/status/jochettino/twisted-tales/ci.yml?branch=main&label=Build)](https://github.com/jochettino/twisted-tales/actions/workflows/ci.yml)

Twisted Tales is a social mystery card game of lateral thinking where players work together like detectives to reconstruct bizarre, macabre, or seemingly impossible situations.

New to the game? See [How to Play](docs/how-to-play.md).

This repository contains a small web prototype that shows one card at a time, in English or Spanish.

## Run it locally

Requirements: [Go](https://go.dev/dl/) 1.22 or newer.

```sh
go run .
```

Then open <http://localhost:8080>. To use another port, set the `PORT` environment variable:

```sh
PORT=3000 go run .
```

Cards are embedded into the binary, so restart the server after adding or editing a card.

## Tech stack

- **Go**, standard library only (`net/http`, `html/template`, `embed`, `encoding/json`), no external dependencies.
- **Cards** are JSON files in `resources/`, named `xxx-smalltitle-LANG.json` (e.g. `001-hiccups-EN.json`), with `title`, `prompt` and `solution` fields. Every card must exist in every language (EN and ES).
- **Frontend** is server-rendered HTML and CSS in `templates/card.html`, with no JavaScript.
- **User state** (selected language and current card) is remembered in browser cookies, so the server stores nothing. Reopening the site resumes where you left off. Cookies are marked `Secure` when served over https, and still work on plain `http://localhost`.

## Testing

```sh
go test ./...   # unit and HTTP route tests
go vet ./...    # static checks
gofmt -l .      # lists files that need formatting (no output means all good)
```

The tests cover card loading (ordering, language suffixes, EN/ES parity and error cases), page navigation, and the HTTP routes (redirect, 404s, solution hidden by default, language switch, help section, saved language and card cookies). Because they load the real files in `resources/`, adding a card without its translation makes them fail.

For a quick visual check, run `go run .` and open <http://localhost:8080>.
