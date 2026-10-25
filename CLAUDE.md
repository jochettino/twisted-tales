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
- Routes: `/` redirects to a card (see user state below); `/cards/{n}` renders the card. The language is chosen with `?lang=ES`/`?lang=EN` (default EN) and carried on every link. The solution is shown only with `?solution=1`, so navigating Back/Next always lands on the prompt. The UI is server-rendered HTML; the help and About sections are `<details>` elements. A small inline script at the bottom of `templates/card.html` (progressive enhancement, no libraries) intercepts clicks on links marked `data-nav`, fetches the same card as JSON from `GET /api/cards/{n}` (same `?lang=` and `?solution=1` as the page; `cardForRequest` in `main.go` serves both routes, so the API also saves the cookies) and fills it into the existing elements by id with `textContent` (never HTML), using `pushState` for Back/Next and `replaceState` for the solution toggle. The API returns only data (`number`, `total`, `title`, `prompt`, `solution` only when requested, `prev`/`next` or null; see `cardData` in `cards.go`), with no markup or UI texts, and a JSON 404 `{"error": ...}` for a missing card. The UI labels the script needs (and the current/other language) are embedded once in `data-*` attributes on `<main id="app">`, because the language only changes with a full reload (the language link has no `data-nav`). The page is always fully server-rendered, so the links work without JavaScript, and a failed fetch falls back to a normal navigation. The solution block is always in the HTML but `hidden` and empty unless `?solution=1`, so a solution is never sent unrequested. When changing the markup, keep the element ids, `data-*` attributes and `pageURL` in the script in sync with the template; disabled Back/Next are `<a>` elements without `href` (class `disabled`, `aria-disabled`) that the script re-enables.
- User state lives in the browser, never on the server: viewing a card sets two cookies, `lang` and `card` (1 year, `HttpOnly`, `SameSite=Lax`, and `Secure` only when the request is https, directly or via `X-Forwarded-Proto` as on Vercel, so plain-http localhost still works; see `setCookie` in `main.go`). `/` redirects to the saved card (falling back to 1 if it is missing or out of range). Language precedence is explicit `?lang=` > `lang` cookie > default (`resolveLang` in `cards.go`). The solution toggle is deliberately not saved.
- Back/Next do not wrap around; they are disabled at the first and last card.

## UI principles

- **Mobile first.** Nearly every player uses the app on a phone, in the phone's browser, so design and test for a narrow touch screen first; desktop only has to not break. Keep the single narrow column, tap targets of at least 3rem, no hover-only interactions and no horizontal scrolling, and respect the safe-area insets. Check layout changes at roughly 360–400 px width.
- **Look and feel.** The look imitates a physical Black Stories card: a black background, white borders and text, and red only as an accent (the title, the drips, the solution divider, the main button). Keep the card as a bordered rectangle that reads like a real card.

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

## Working in parallel

Several agents may work on this repo at the same time, so follow this to avoid collisions:

- Never edit in the shared checkout. Do all work in your own git worktree on its own branch (e.g. `EnterWorktree`), branched from `origin/main`. Local `main` may hold unpushed commits; don't assume it matches `origin/main`.
- Never commit or push to `main`, force-push, or merge. Push your branch and open a draft PR; the user reviews and merges. A push to `main` deploys to production (see Deployment), so only the user does that.
- Stay in your assigned scope. Shared files (`CLAUDE.md`, `templates/card.html`, `template.go`, `cards.go`) are conflict hotspots: touch them only if the task needs it, and keep edits small and targeted. If the task seems to need a file outside your scope, say so instead of editing it.
- Card numbers: the numeric filename prefix defines the order, so two agents adding cards will pick the same next number. Use only the number range you were assigned. If none was given, ask before adding cards. Every card needs both an EN and an ES file.
- Ports: never assume 8080 is free. When running the server, pick a distinct `PORT` (e.g. `PORT=18080 go run .`) and stop the server when done.
- Don't use bare `git stash` (the stash is shared across worktrees); commit work in progress instead. Keep temporary files out of the repo.
- Before opening a PR, run `go test ./...`, `go vet ./...` and `gofmt -l .`, and rebase on the latest `origin/main`. In the PR description, say if the change affects the build or runtime (see Deployment).
- Branches merge one at a time. If your PR conflicts after another merged, rebase and resolve; don't overwrite the other change.

## Git

Commits are authored as `jochettino` with a GitHub noreply email (set in the global git config). Don't substitute a personal email.
