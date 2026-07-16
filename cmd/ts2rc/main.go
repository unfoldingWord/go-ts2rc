// Command ts2rc converts a translationStudio repository to a Resource Container (RC).
//
// Usage:
//
//	ts2rc [flags] <tsRepoDir> <rcOutDir>
//	ts2rc --tw-source /path/to/en_tw <tsRepoDir> <rcOutDir>
//
// Flags:
//
//	--tw-source <dir>   Path to a canonical Translation Words RC repo (e.g. en_tw)
//	                    for kt/names/other category lookup when converting tw projects.
//
// A conversion report is written to <rcOutDir>.report.json. Exit code is 0 when
// the conversion produced a usable RC repo, 1 otherwise.
//
// Pipeline with go-rc2sb (ts → RC → SB):
//
//	ts2rc <tsDir> <rcDir>
//	rc2sb <rcDir> <sbDir>
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	ts2rc "github.com/unfoldingWord/go-ts2rc"
)

func main() {
	twSource := flag.String("tw-source", "", "path to a canonical tw RC repo (e.g. en_tw clone) for kt/names/other category lookup")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `ts2rc — convert a translationStudio repo to a Resource Container repo

Usage:
  ts2rc [flags] <tsRepoDir> <rcOutDir>

A conversion report is written to <rcOutDir>.report.json. Exit code is 0 when
the conversion produced a usable RC repo, 1 otherwise.

Pipeline (ts → RC → SB):
  ts2rc [--tw-source en_tw] <tsDir> <rcDir>
  rc2sb [--usfm ...] [--payload ...] <rcDir> <sbDir>

Flags:
`)
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	inDir, outDir := flag.Arg(0), flag.Arg(1)

	rep := ts2rc.Convert(context.Background(), inDir, outDir, ts2rc.Options{TWSourceDir: *twSource})

	reportPath := filepath.Clean(outDir) + ".report.json"
	if err := ts2rc.WriteReport(rep, reportPath); err != nil {
		fmt.Fprintf(os.Stderr, "writing report: %v\n", err)
	}

	if !rep.OK {
		fmt.Fprintf(os.Stderr, "FAILED %s: %s\n", inDir, rep.Error)
		os.Exit(1)
	}
	fmt.Printf("OK %s -> %s (class=%s, v%d, warnings=%d)\n", inDir, outDir, rep.Class, rep.Version, len(rep.Warnings))
}
