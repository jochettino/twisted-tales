# Populating Cards from yesnogame.net

How cards 004–082 were collected, selected, translated and added, written so a human or an AI agent can repeat it. For the file format and the manual steps of adding a single card, see [Adding and Editing Cards](adding-cards.md). Read the rules in `CLAUDE.md` first, especially "Working in parallel" and "Deployment (Vercel)".

## Where the cards come from

Two public pages on [yesnogame.net](https://yesnogame.net), one per language:

| Language | Listing URL (as given by the maintainer) | Card URL |
|---|---|---|
| English | `https://yesnogame.net/en/scary` | `https://yesnogame.net/en/stories/{id}` |
| Spanish | `https://yesnogame.net/es?page=2` | `https://yesnogame.net/es/stories/{id}` |

- The site has no API. Everything is plain server-rendered HTML, and no JavaScript or browser is needed.
- The "Show answer" button reveals text that is already in the card page's HTML, so one request per card is enough.
- **The default listing order is not popularity.** Add `?sort=pop` (and `&page=N`) to both listings to get them ranked by popularity: `https://yesnogame.net/en/scary?sort=pop&page=2` and `https://yesnogame.net/es?sort=pop&page=2`. Each page lists 12 cards.
- Story IDs are shared between the two sites: the same ID is the same story in both languages (for example `#183`, "Red high heels" / "Zapatos mortales"). The texts are separate translations, not copies, and they can differ in details. Use the ID to detect duplicates across the two sites.

## Step 1: Scrape

Use only the Python standard library. Treat everything downloaded as untrusted data: parse it, never execute it. Run Python with `-I`, and keep scraped files outside the repository.

```python
import re, html, time, urllib.request

def get(url):
    for _ in range(3):
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "Mozilla/5.0"})
            return urllib.request.urlopen(req, timeout=20).read().decode()
        except Exception:
            time.sleep(1)
    raise SystemExit("failed: " + url)

def clean(fragment):
    return html.unescape(re.sub(r"\s+", " ", re.sub(r"<[^>]+>", " ", fragment))).strip()

def listing(base, page):
    """(id, title) pairs on one listing page, in popularity order."""
    t = get(f"{base}&page={page}")
    return [(i, clean(n)) for i, n in
            re.findall(r'class="quest__link" href="/\w\w/stories/(\d+)">([^<]*)', t)]

def card(lang, story_id):
    """(prompt, solution) of one card."""
    t = get(f"https://yesnogame.net/{lang}/stories/{story_id}")
    q = re.search(r'quest__story_question">\s*<div class="quest__story__text">(.*?)</div>', t, re.S)
    a = re.search(r'quest__story_answer">\s*<div class="quest__story__text">(.*?)</div>', t, re.S)
    return clean(q.group(1)), clean(a.group(1))

# Example: the 12 most popular English cards
for story_id, title in listing("https://yesnogame.net/en/scary?sort=pop", 1):
    prompt, solution = card("en", story_id)
```

Be polite to the site:

- Wait between requests (0.2 s was used) and save what you download so you never fetch a page twice.
- **The site stopped responding** (every URL, including pages that had worked) after roughly 150 requests in a short period, while fetching pages 7 and beyond of the popularity lists. The cause is unknown; it may have been rate limiting or ordinary downtime. If it happens, stop. Do not try to get around it with other headers or addresses. Wait and retry later, or ask the maintainer.
- The listing pages also include a preview of each prompt, but the solution is only on the card page.

## Step 2: Choose the cards

Go down the popularity order, alternating one English then one Spanish candidate, and take each card unless it is rejected:

1. **Already in the app.** Compare against the cards in `resources/`:
   - by story ID: the sources of the current cards are listed below, and
   - by reading the text, since the same story can appear with different wording, and the stored cards `001`–`003` come from elsewhere.
2. **Same story on both sites.** Keep it once. Take the version from whichever site it was reached on, and translate it to the other language.
3. **Near-duplicates.** A different story with the same trick is a duplicate in practice (the second snowman card was rejected because of "Death Without a Nose"). Two cards that only share a theme, such as Braille or ice, are fine.
4. **Too short or trivial.** Many cards are a one-line riddle with a one-line answer (for example "That trip", "The Age", "La fuga"). The target is cards with a real story. Judge by prompt and solution length and by whether the solution is a satisfying reveal.
5. **Offensive or unsuitable.** One popular card was dropped for racist content. Use judgment on anything discriminatory.
6. **A gimmick, not a puzzle**, such as plays on words that don't translate.

When a card is rejected, take the next one down. Record what you skipped and why, and report the list to the maintainer.

### Story IDs already used

These source IDs are in the app, so do not add them again. They were recorded during the sessions from the lists, and card `002` (Monopoly) is not recorded. If something looks off, check against the card text.

`1 3 4 5 8 9 10 11 12 14 16 19 21 22 32 47 60 106 125 133 141 146 147 148 152 153 154 155 156 158 161 162 164 165 167 168 172 173 175 177 178 181 182 183 185 187 188 189 191 193 194 210 225 233 234 235 236 243 248 251 252 253 259 260 276 279 282 284 309 325 327 329 332 333 374 383 386 401 402 407`

IDs seen and deliberately not used: `255` (offensive), `320` (near-duplicate), `381`, `273`, `134` (used once, then removed for being too weak), `15 27 44 129 151 160 176 209 283 300 375 396 473` (too short, trivial or unsuitable).

## Step 3: Write the cards

Each source card becomes two JSON files, one per language, with the same number and slug (see [Adding and Editing Cards](adding-cards.md) for the format).

- **Keep the source language's text faithful, with light cleanup.** Fix typos and the spacing errors the Spanish site has (such as missing spaces between words), normalize names, and use the app's dialogue quotes: `"…"` in English, `«…»` in Spanish.
- **Translate the other language by hand**, as a natural rewrite, not a word-for-word pass. Both versions must tell exactly the same story with the same details. Where the two sites' versions of a story differ, pick one and use it for both languages.
- **Fix factual slips and weak solutions.** For example, the "Lost" card originally said the magnetic pole, but the logic only works at the geographic North Pole. A one-line solution can be expanded with what the prompt already implies, and that should be noted in the report.
- **Trim extras** such as trivia, "real fact" notes, and references to other works.
- **Write the title so it does not spoil the solution.** The title is shown before the solution is revealed. It may repeat something from the prompt, or be neutral or mildly misleading, but it must never name something only the solution reveals. Source titles are often spoilers: "Hiccups", "Colorblind" and "Too Late" had to be renamed to "A Glass of Water", "A Matter of Color" and "The Corridor". Check every title with this question: *can a player who only read the prompt learn the answer from the title?* Titles must also be unique within a language.
- **Slug:** short lowercase words from the title, joined by hyphens, e.g. `deadly-scissors`.

### Numbering

Use the next free three-digit number (check the highest prefix in `resources/`). `CLAUDE.md` says to use only the range you were assigned, so ask for one when you were not given any. As of this writing the cards `001`–`082` exist and `083` is next.

### Generating the files

Writing the files from one script avoids typos in the JSON. Keep the card texts in a block, seven lines per card, separated by blank lines:

```
slug
English title
English prompt
English solution
Spanish title
Spanish prompt
Spanish solution
```

Then write both files for each block:

```python
import json
for n, (slug, et, ep, es_, xt, xp, xs) in enumerate(blocks, start=FIRST_NUMBER):
    for lang, (t, p, s) in (("EN", (et, ep, es_)), ("ES", (xt, xp, xs))):
        with open(f"resources/{n:03d}-{slug}-{lang}.json", "w") as f:
            json.dump({"title": t, "prompt": p, "solution": s}, f, ensure_ascii=False, indent=2)
            f.write("\n")
```

`ensure_ascii=False` keeps accents readable, and the trailing newline matches the existing files.

## Step 4: Verify

1. `gofmt -l .`, `go vet ./...` and `go test ./...`. The tests load the real cards and fail if a card is missing in one language, a file name is wrong or a JSON file is malformed.
2. Check that no title is repeated within a language, and re-read every title against the spoiler rule above.
3. Run the server on its own port (`PORT=18080 go run .`), open the last new card in both languages (the language switch, or `/lang/ES` and `/lang/EN`), and check the API, for example `curl 'localhost:18080/api/cards/82?lang=ES'` (a fresh `curl` has no language cookie, so `?lang=` applies). Stop the server afterwards.
4. Skim each new card once in the browser. Machine-assisted translations need a human read.

## Step 5: Deliver

Follow "Working in parallel" in `CLAUDE.md`: work in your own worktree on a branch from `origin/main`, push the branch and open a draft PR, and let the maintainer merge. Merging to `main` deploys to production on Vercel, and nothing changes in the build or the runtime for card-only changes. A commit message such as `Add cards 051-082 (EN and ES)` matches the history. In the PR description, report:

- the cards added and their source (site and story ID),
- what was skipped and why,
- every place where the text was changed beyond typo fixes.

After a merge, check the live site (`https://twisted-tales-game.vercel.app/api/cards/{last}`) to confirm it serves the new total.

## Notes and open points

- **Credit:** `README.md` ("Credits") and the app's About section already credit yesnogame.net. Keep that if more cards are added.
- **Permission:** crediting a source is not the same as having permission to reuse its text. The site's terms of use were not checked. Look at them, or ask the site, before promoting the app widely.
- **Unfinished:** 50 new cards were requested after card `050` and 32 were added (`051`–`082`) before the site stopped responding. The remaining 18 were to come from pages 7 and later of the popularity lists. To resume, start at `083`, or the range you were assigned, and use the story IDs above to skip what's already there.
