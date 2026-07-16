# go-ts2rc

Go library for converting [translationStudio](https://www.unfoldingword.org/ts) (ts) project repositories to [Resource Container](https://resource-container.readthedocs.io/) (RC) format.

Output RC repositories are compatible with [go-rc2sb](https://github.com/unfoldingWord/go-rc2sb) for conversion to [Scripture Burrito](https://docs.burrito.bible/) (SB):

```text
ts repo  →  go-ts2rc  →  RC repo  →  go-rc2sb  →  SB repo
```

## Installation

```bash
go get github.com/unfoldingWord/go-ts2rc
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    ts2rc "github.com/unfoldingWord/go-ts2rc"
)

func main() {
    ctx := context.Background()

    rep := ts2rc.Convert(ctx, "/path/to/ts-repo", "/path/to/rc-output", ts2rc.Options{})
    if !rep.OK {
        log.Fatal(rep.Error)
    }

    fmt.Printf("Converted class=%s package_version=%d warnings=%d\n",
        rep.Class, rep.Version, len(rep.Warnings))
}
```

### TW category lookup

For Translation Words projects, pass a canonical TW RC directory (for example `en_tw`) so word slugs are placed under `bible/kt/`, `bible/names/`, or `bible/other/`:

```go
opts := ts2rc.Options{
    TWSourceDir: "/path/to/en_tw",
}
rep := ts2rc.Convert(ctx, "/path/to/ts-tw-repo", "/path/to/rc-output", opts)
```

If `TWSourceDir` is empty or a slug is not found, articles are written under `bible/other/` and a warning is recorded.

### CLI tool

A simple CLI wrapper is available at `cmd/ts2rc/`:

```bash
go run ./cmd/ts2rc /path/to/ts-repo /path/to/rc-output

# With TW category lookup
go run ./cmd/ts2rc --tw-source /path/to/en_tw /path/to/ts-tw /path/to/rc-output
```

A conversion report is written to `<rcOutDir>.report.json`.

### End-to-end: ts → SB via go-rc2sb

```bash
# 1) ts → RC
go run ./cmd/ts2rc --tw-source /path/to/en_tw /path/to/ts-repo /path/to/rc-repo

# 2) RC → SB (https://github.com/unfoldingWord/go-rc2sb)
rc2sb /path/to/rc-repo /path/to/sb-output

# Optional go-rc2sb flags:
#   --usfm /path/to/usfm-dir     localized book names for TSV repos
#   --payload /path/to/en_tw     TWL payload (when converting TWL RCs)
```

## API

### `Convert(ctx, inDir, outDir, opts) *Report`

Converts a translationStudio repository to RC format.

- `ctx` — Context for cancellation
- `inDir` — Path to the ts repository (must contain `manifest.json`)
- `outDir` — Path where the RC output will be written
- `opts` — Conversion options (see Options below)

Returns a `Report`. Check `Report.OK` for success. On hard failure, incomplete output is removed and `Report.Error` is set. Non-fatal issues appear in `Report.Warnings`.

### Options

```go
type Options struct {
    // TWSourceDir is a canonical Translation Words RC path (e.g. en_tw)
    // used to place word articles under bible/{kt|names|other}/.
    TWSourceDir string
}
```

### Report

```go
type Report struct {
    Source   string         // input ts directory
    Output   string         // output RC directory
    Class    string         // bible | obs | tq | tn | obs-tq | obs-tn | tw
    Version  int            // package_version
    OK       bool
    Error    string         // set when OK is false
    Warnings []string
    Stats    map[string]int // e.g. chunks, rows, stories, words
}
```

## Supported conversion classes

| ts type + project | RC identifier | RC subject | Content |
|---|---|---|---|
| text + book | resource.id (`ulb`/`udb`/`reg`/…) | Bible | `{NN}-{BOOK}.usfm` |
| text + obs | `obs` | Open Bible Stories | `content/{NN}.md`, `content/front/title.md` |
| tq + book | `tq` | TSV Translation Questions | `tq_{BOOK}.tsv` |
| tn + book | `tn` | TSV Translation Notes | `tn_{BOOK}.tsv` |
| tq + obs | `obs-tq` | TSV OBS Translation Questions | `tq_OBS.tsv` |
| tn + obs | `obs-tn` | TSV OBS Translation Notes | `tn_OBS.tsv` |
| tw + bible | `tw` | Translation Words | `bible/{kt\|names\|other}/{word}.md` |

Supports translationStudio `package_version` 3 and 5–8 (v3 and v5+ manifest schemas).

## Error handling

- Missing or invalid `manifest.json` → `Report.OK == false` with error
- Unrecognized type/project combination → error listing the problem
- Context cancellation is checked at the start and after content conversion
- Bad chunks / empty frames → warn and continue (fail-safe); conversion still succeeds when usable content remains
- Hard conversion failure removes the incomplete `outDir`

## Building

```bash
go build ./...
go build -o ts2rc ./cmd/ts2rc
```

## Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run only unit tests for packages
go test ./books/... ./ts/... ./rc/...
```

## Architecture

```text
go-ts2rc/
├── convert.go              # Public Convert() function
├── options.go              # Options, Report, Result types
├── license.go              # Default CC BY-SA 4.0 LICENSE.md
├── chunks.go               # Shared ts chunk directory helpers
├── bible.go                # Bible / USFM conversion
├── obs.go                  # Open Bible Stories conversion
├── tsv.go                  # TN / TQ (book + OBS) conversion
├── tw.go                   # Translation Words conversion
├── cmd/ts2rc/
│   └── main.go             # CLI wrapper
├── books/
│   └── books.go            # Bible book codes, sort, USFM numbers
├── ts/
│   └── manifest.go         # translationStudio manifest.json parsing
└── rc/
    └── manifest.go         # RC 0.2 manifest.yaml types and writer
```

## License

See [LICENSE](LICENSE) for details.
