# Adding and Editing Cards

Cards are plain JSON files in `resources/`. They are embedded into the Go binary at build time, so there is no database and no admin screen: a card change is a file change.

## File format

Each card needs one file per language. Today the languages are English (`EN`) and Spanish (`ES`).

```
resources/005-top-students-EN.json
resources/005-top-students-ES.json
```

The file name is `NNN-smalltitle-LANG.json`:

- `NNN` is a three-digit number. Cards are ordered by file name, so this number defines the order in the app.
- `smalltitle` is a short lowercase slug with words separated by hyphens.
- `LANG` is the language code in capitals: `EN` or `ES`.

The content is a JSON object with exactly these fields:

```json
{
  "title": "A Glass of Water",
  "prompt": "A man enters a bar and asks for a glass of water. The bartender pulls out a shotgun and points it directly at him. The man says 'Thank you' and walks out.",
  "solution": "The man had the hiccups and wanted water to cure them. The bartender realised this and scared the hiccups out of him with the shotgun instead. The man, cured, thanked him and left."
}
```

- `title`: shown as the card heading. It is visible before the solution, so it must not give the answer away.
- `prompt`: the situation the players try to solve. The Game Master reads it aloud.
- `solution`: revealed with "Show solution". Write it as a full explanation of what happened.

Use plain text only. Escape double quotes inside a value as `\"`, or use single quotes as in the example.

## Adding a new card

1. **Check it isn't a duplicate.** Skim the existing prompts in `resources/` for the same or a very similar story.
2. **Pick the next number.** Look at the highest prefix in `resources/` and add one (after `050`, use `051`).
3. **Create both language files** with the same number and slug, for example `051-lighthouse-EN.json` and `051-lighthouse-ES.json`. The two files must tell the same story. The number and slug must match exactly, because the app pairs the languages by that part of the name.
4. **Run the tests**: `go test ./...`. The app will refuse to start, and the tests will fail, if a card exists in one language only, if a file name has an unknown language suffix, or if a JSON file is malformed.
5. **Restart the server** (`go run .`) and open the card, in both languages (`?lang=ES`), to check how it reads. Cards are embedded in the binary, so changes only show after a restart.
6. **Commit and push to `main`.** CI runs the tests when card JSON changes, and Vercel redeploys from `main`.

## Editing an existing card

1. Edit the `title`, `prompt` or `solution` in **both** language files, so the languages stay in sync.
2. Run `go test ./...`, restart the server and check the card.
3. Commit and push.

If you change a title because it gave away the solution, change it in both languages too.

## Renaming, reordering and removing cards

- To **rename** a card, rename both language files so the number and slug still match. Use `git mv` to keep the history.
- To **remove** a card, delete both language files.
- To **reorder** a card, change its number prefix in both files.

Keep in mind that the card numbers in URLs (`/cards/12`) and in the saved `card` cookie are positions in the sorted list, not the file prefix. Inserting, removing or reordering cards shifts the cards after that point. Existing links and players' saved places will then point at a different card. Appending new cards at the end avoids that.

## Adding a language

Adding a third language is not just a data change. It needs code changes (the language switch in `newPage` assumes exactly two languages, and the labels in `template.go`) plus a translated file for every existing card. See the Architecture section of `CLAUDE.md`.
