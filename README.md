# Twisted Tales

[![Vercel status](https://img.shields.io/website?url=https%3A%2F%2Ftwisted-tales-game.vercel.app&up_message=OK&down_message=KO&up_color=brightgreen&down_color=red&label=Vercel)](https://twisted-tales-game.vercel.app) [![Build status](https://img.shields.io/github/actions/workflow/status/jochettino/twisted-tales/ci.yml?branch=main&label=Build)](https://github.com/jochettino/twisted-tales/actions/workflows/ci.yml)

Twisted Tales is a social mystery card game of lateral thinking where players work together like detectives to reconstruct bizarre, macabre, or seemingly impossible situations.

New to the game? See [How to Play](docs/how-to-play.md).

Want to add or change cards? See [Adding and Editing Cards](docs/adding-cards.md). To add many cards at once from yesnogame.net, see [Populating Cards](docs/populating-cards.md).

This repository contains a small web prototype that shows one card at a time, in English or Spanish. The UI is designed mobile first, for players using a phone.

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
- **Cards** are JSON files in `resources/`, named `xxx-smalltitle-LANG.json` (e.g. `001-glass-of-water-EN.json`), with `title`, `prompt` and `solution` fields. Every card must exist in every language (EN and ES).
- **Frontend** is server-rendered HTML and CSS in `templates/card.html`. A small inline script (no libraries) makes Back, Next and Show/Hide solution fetch the card as JSON from `GET /api/cards/{n}?solution=1` and update only the card instead of reloading the page; without JavaScript they are ordinary links.
- **User state** (selected language and current card) is remembered in browser cookies, so the server stores nothing. Reopening the site resumes where you left off. Cookies are marked `Secure` when served over https, and still work on plain `http://localhost`.

## Testing

```sh
go test ./...   # unit and HTTP route tests
go vet ./...    # static checks
gofmt -l .      # lists files that need formatting (no output means all good)
```

The tests cover card loading (ordering, language suffixes, EN/ES parity and error cases), page navigation, and the HTTP routes (redirect, 404s, solution hidden by default, language switch, help section, saved language and card cookies). Because they load the real files in `resources/`, adding a card without its translation makes them fail.

For a quick visual check, run `go run .` and open <http://localhost:8080>.

## Credits

Many of the cards were adapted from [yesnogame.net](https://yesnogame.net), which publishes these situation puzzles in several languages. Where a puzzle was available in only one language there, the other language is our own translation. Thanks to the authors and community behind that site.
