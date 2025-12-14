# Project rules

## General

- Please ask questions before doing any changes if you have any doubts about anything.

## Testing

- Always use `just test` command for testing.
- Use standard `testing` package.
- Place tests next to code as `*_test.go`.
- Use `TestXxx` naming for tests.

## git

- Always check staged and unstaged changes before doing any work to have a clear context.
- Don't stage/unstage any changes and don't do any commits until explicitly asked.

## DB

- Always consider performance and complexity and try to use existing or create new indexes.

## Docs

- In any Markdown file please consider the max line length equal 120 (excluding tables, long links or code blocks, etc.)
- After doing any changes in project, check that any existing docs must be actualized:
  - `AGENTS.md`
  - `README.md`
  - code comments
- New Markdown doc files must be named in the uppercase with underscores, e.g. `SOME_DOCUMENT.md`.

## Security

- Never log any sensitive date.

## Logging

- For logging, use `log/slog` with structured fields.

## Error handling

- Wrap errors with `fmt.Errorf` and use `errors.Join` when aggregating.

## Code conventions

- Follow defined code style rules (see `.golangci.yaml` and `.editorconfig`).
