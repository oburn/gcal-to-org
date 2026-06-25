# AGENTS.md

## Project Overview
- `gcal-to-org` is a Go CLI that reads the user's primary Google Calendar and writes Org mode output.
- The active code is in the root Go files: `main.go`, `auth.go`, `generate.go`, and `room.go`.
- `generate` creates an Org file from calendar events. `room` finds upcoming meetings without rooms and suggests available rooms.
- Gradle/Kotlin files exist, but `src/main/kotlin/gcal2org/Runner.kt` is currently skeletal; treat the Go CLI as the source of truth unless asked otherwise.

## Common Commands
- `make test` or `go test -v ./...` - compile and run Go tests.
- `make run ARGS="generate /tmp/calendar.org"` - run the CLI through Make.
- `go run . --store /tmp/gcal-to-org generate /tmp/calendar.org` - run directly with an isolated token store.
- `make build` or `go build .` - build the binary.
- `make tidy` - update `go.mod` and `go.sum` after dependency changes.

## Credentials and Local State
- OAuth client values live in `secrets.go`; start from `secrets.go.template` if needed.
- Do not commit real client IDs, client secrets, OAuth tokens, or generated calendar output.
- Use `--store` to point OAuth token storage at a temporary or test-specific directory when running locally.
- Commands that hit Google Calendar may open a browser and require interactive authorization.

## Coding Notes
- Keep Go code in package `main` and prefer small helper functions matching the current style.
- Run `gofmt` on modified Go files.
- Preserve CLI behavior built with `urfave/cli/v2`; add flags and subcommands through the existing command factory pattern.
- Calendar logic should remain testable by keeping formatting/filtering helpers separate from Google API calls.
- Room filtering is currently Sydney office specific (`SYD 363 George St`, levels 28/30); avoid broadening it unless requested.

## Validation
- Always run `go test -v ./...` after Go changes.
- If changing module dependencies, run `make tidy` and include any intentional `go.mod`/`go.sum` updates.
