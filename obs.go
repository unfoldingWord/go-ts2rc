package ts2rc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/unfoldingWord/go-ts2rc/rc"
	"github.com/unfoldingWord/go-ts2rc/ts"
)

// stripEmphasis removes wrapping _ or * from a reference line.
func stripEmphasis(s string) string {
	return strings.Trim(strings.TrimSpace(s), "_*")
}

// convertOBS assembles RC content/{NN}.md story files from ts frame chunks.
func convertOBS(inDir, outDir string, m *ts.Manifest, rep *Report) error {
	contentDir := filepath.Join(outDir, "content")
	if err := os.MkdirAll(filepath.Join(contentDir, "front"), 0o755); err != nil {
		return err
	}

	bookTitle := readTextFile(filepath.Join(inDir, "front", "title.txt"))
	if bookTitle == "" {
		bookTitle = readTextFile(filepath.Join(inDir, "00", "title.txt"))
	}
	if bookTitle == "" {
		bookTitle = m.ProjectName
	}
	if bookTitle == "" {
		bookTitle = "Open Bible Stories"
		rep.Warnf("no OBS title found; using %q", bookTitle)
	}
	if err := os.WriteFile(filepath.Join(contentDir, "front", "title.md"), []byte(bookTitle+"\n"), 0o644); err != nil {
		return err
	}

	dirs, err := listChunkDirs(inDir)
	if err != nil {
		return err
	}
	stories := 0
	for _, d := range dirs {
		if d.Num == 0 || d.Num > 50 {
			continue // 00/ is the title dir; OBS has stories 01-50
		}
		files, err := listChunkFiles(d.Path)
		if err != nil {
			rep.Warnf("story %s: %v", d.Name, err)
			continue
		}

		storyNum := fmt.Sprintf("%02d", d.Num)
		var b strings.Builder
		title := readTextFile(filepath.Join(d.Path, "title.txt"))
		if title == "" {
			title = fmt.Sprintf("%s %d", bookTitle, d.Num)
			rep.Warnf("story %s: no title.txt; using %q", d.Name, title)
		}
		fmt.Fprintf(&b, "# %s\n", title)

		frames := 0
		for _, f := range files {
			if f.Num < 0 {
				continue // title.txt / reference.txt
			}
			text := readTextFile(f.Path)
			if text == "" {
				rep.Warnf("story %s frame %s: empty, skipped", d.Name, f.Base)
				continue
			}
			frameNum := fmt.Sprintf("%02d", f.Num)
			fmt.Fprintf(&b, "\n![OBS Image](https://cdn.door43.org/obs/jpg/360px/obs-en-%s-%s.jpg)\n\n%s\n", storyNum, frameNum, text)
			frames++
		}
		if frames == 0 {
			rep.Warnf("story %s: no frames, skipped", d.Name)
			continue
		}
		if ref := stripEmphasis(readTextFile(filepath.Join(d.Path, "reference.txt"))); ref != "" {
			fmt.Fprintf(&b, "\n_%s_\n", ref)
		}
		if err := os.WriteFile(filepath.Join(contentDir, storyNum+".md"), []byte(b.String()), 0o644); err != nil {
			return err
		}
		stories++
	}
	if stories == 0 {
		return fmt.Errorf("no stories found")
	}
	rep.Stats["stories"] = stories

	man := baseManifest(inDir, m, rep)
	man.DublinCore.Format = "text/markdown"
	man.DublinCore.Identifier = "obs"
	man.DublinCore.Subject = "Open Bible Stories"
	man.DublinCore.Title = bookTitle
	man.DublinCore.Type = "book"
	man.Projects = []rc.Project{{
		Title:      bookTitle,
		Identifier: "obs",
		Sort:       0,
		Path:       "./content",
	}}
	return rc.WriteManifest(outDir, man)
}
