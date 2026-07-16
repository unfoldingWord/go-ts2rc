package ts2rc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unfoldingWord/go-ts2rc/rc"
	"github.com/unfoldingWord/go-ts2rc/ts"
)

// twCategory resolves a word slug to kt/names/other using a canonical tw repo
// (e.g. a local clone of unfoldingWord/en_tw) as the lookup; falls back to "other".
func twCategory(twSourceDir, slug string) (string, bool) {
	if twSourceDir == "" {
		return "other", false
	}
	for _, cat := range []string{"kt", "names", "other"} {
		if _, err := os.Stat(filepath.Join(twSourceDir, "bible", cat, slug+".md")); err == nil {
			return cat, true
		}
	}
	return "other", false
}

// convertTW converts a ts translationWords repo (numbered dirs of {word}.txt
// JSON chunks) into the RC bible/{kt|names|other}/{word}.md layout.
func convertTW(inDir, outDir string, m *ts.Manifest, opts Options, rep *Report) error {
	dirs, err := listChunkDirs(inDir)
	if err != nil {
		return err
	}
	words, unmatched := 0, 0
	for _, d := range dirs {
		files, err := listChunkFiles(d.Path)
		if err != nil {
			rep.Warnf("group %s: %v", d.Name, err)
			continue
		}
		for _, f := range files {
			if f.Num >= 0 || f.Base == "title" || f.Base == "reference" {
				continue
			}
			entries, err := readTitleBodyChunk(f.Path)
			if err != nil {
				rep.Warnf("word %s: not a valid JSON chunk (%v), skipped", f.Base, err)
				continue
			}
			var b strings.Builder
			for _, e := range entries {
				title := strings.TrimSpace(e.Title)
				body := strings.TrimSpace(e.Body)
				if title == "" && body == "" {
					continue
				}
				if b.Len() > 0 {
					b.WriteString("\n")
				}
				fmt.Fprintf(&b, "# %s\n\n%s\n", title, body)
			}
			if b.Len() == 0 {
				rep.Warnf("word %s: empty article, skipped", f.Base)
				continue
			}
			cat, matched := twCategory(opts.TWSourceDir, f.Base)
			if !matched {
				unmatched++
			}
			catDir := filepath.Join(outDir, "bible", cat)
			if err := os.MkdirAll(catDir, 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(catDir, f.Base+".md"), []byte(b.String()), 0o644); err != nil {
				return err
			}
			words++
		}
	}
	if words == 0 {
		return fmt.Errorf("no word articles found")
	}
	rep.Stats["words"] = words
	if unmatched > 0 {
		rep.Warnf("%d word slugs not found in canonical tw source; placed under bible/other/", unmatched)
	}

	title := m.ProjectName
	if title == "" || strings.EqualFold(title, "translationwords") {
		title = "translationWords"
	}
	man := baseManifest(inDir, m, rep)
	man.DublinCore.Format = "text/markdown"
	man.DublinCore.Identifier = "tw"
	man.DublinCore.Subject = "Translation Words"
	man.DublinCore.Title = title
	man.DublinCore.Type = "dict"
	man.Projects = []rc.Project{{
		Title:      title,
		Identifier: "bible",
		Sort:       0,
		Path:       "./bible",
	}}
	return rc.WriteManifest(outDir, man)
}
