package ts2rc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/unfoldingWord/go-ts2rc/rc"
	"github.com/unfoldingWord/go-ts2rc/ts"
)

// taManualOrder is the order of the tA manuals in an RC; any other manual follows alphabetically.
var taManualOrder = []string{"intro", "process", "translate", "checking", "audio", "gateway"}

var taManualTitles = map[string]string{
	"intro":     "Introduction to translationAcademy",
	"process":   "Process Manual",
	"translate": "Translation Manual",
	"checking":  "Checking Manual",
	"audio":     "Audio Manual",
	"gateway":   "Gateway Languages Manual",
}

var taVolumeRe = regexp.MustCompile(`^vol\d+$`)

// splitTAName splits a ts tA chunk name, {vol}_{manual}_{slug} (the volume is optional),
// into its manual and article slug.
func splitTAName(base string) (manual, slug string, ok bool) {
	name := base
	if vol, rest, found := strings.Cut(base, "_"); found && taVolumeRe.MatchString(vol) {
		name = rest
	}
	manual, slug, found := strings.Cut(name, "_")
	if !found || manual == "" || slug == "" {
		return "", "", false
	}
	return manual, slug, true
}

// convertTA converts a ts translationAcademy repo (numbered dirs of
// {vol}_{manual}_{slug}.txt JSON chunks) into the RC {manual}/{slug}/{title,01}.md layout.
func convertTA(inDir, outDir string, m *ts.Manifest, rep *Report) error {
	dirs, err := listChunkDirs(inDir)
	if err != nil {
		return err
	}
	manuals := map[string]bool{}
	articles := 0
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
			manual, slug, ok := splitTAName(f.Base)
			if !ok {
				rep.Warnf("article %s: name is not {vol}_{manual}_{slug}, skipped", f.Base)
				continue
			}
			entries, err := readTitleBodyChunk(f.Path)
			if err != nil {
				rep.Warnf("article %s: not a valid JSON chunk (%v), skipped", f.Base, err)
				continue
			}
			var title string
			var body strings.Builder
			for _, e := range entries {
				t, b := strings.TrimSpace(e.Title), strings.TrimSpace(e.Body)
				switch {
				case t == "" && b == "":
					continue
				case title == "" && body.Len() == 0:
					title = t
				case t != "":
					fmt.Fprintf(&body, "\n\n# %s\n\n", t)
				default:
					body.WriteString("\n\n")
				}
				body.WriteString(b)
			}
			if title == "" && body.Len() == 0 {
				rep.Warnf("article %s: empty, skipped", f.Base)
				continue
			}
			articleDir := filepath.Join(outDir, manual, slug)
			if err := os.MkdirAll(articleDir, 0o755); err != nil {
				return err
			}
			if title != "" {
				if err := os.WriteFile(filepath.Join(articleDir, "title.md"), []byte(title+"\n"), 0o644); err != nil {
					return err
				}
			}
			if err := os.WriteFile(filepath.Join(articleDir, "01.md"), []byte(body.String()+"\n"), 0o644); err != nil {
				return err
			}
			manuals[manual] = true
			articles++
		}
	}
	if articles == 0 {
		return fmt.Errorf("no academy articles found")
	}
	rep.Stats["articles"] = articles

	order := make([]string, 0, len(manuals))
	for _, manual := range taManualOrder {
		if manuals[manual] {
			order = append(order, manual)
			delete(manuals, manual)
		}
	}
	others := make([]string, 0, len(manuals))
	for manual := range manuals {
		others = append(others, manual)
	}
	sort.Strings(others)
	order = append(order, others...)

	title := m.ProjectName
	if title == "" {
		title = "translationAcademy"
	}
	man := baseManifest(inDir, m, rep)
	man.DublinCore.Format = "text/markdown"
	man.DublinCore.Identifier = "ta"
	man.DublinCore.Subject = "Translation Academy"
	man.DublinCore.Title = title
	man.DublinCore.Type = "man"
	for i, manual := range order {
		manualTitle := taManualTitles[manual]
		if manualTitle == "" {
			manualTitle = manual
		}
		man.Projects = append(man.Projects, rc.Project{
			Title:      manualTitle,
			Identifier: manual,
			Sort:       i,
			Path:       "./" + manual,
		})
	}
	return rc.WriteManifest(outDir, man)
}
