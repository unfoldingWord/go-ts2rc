package ts2rc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/unfoldingWord/go-ts2rc/books"
	"github.com/unfoldingWord/go-ts2rc/rc"
	"github.com/unfoldingWord/go-ts2rc/ts"
)

var (
	compactChapterRe = regexp.MustCompile(`\\c(\d)`)
	compactVerseRe   = regexp.MustCompile(`\\v(\d)`)
	chapterMarkerRe  = regexp.MustCompile(`\\c\s+\d`)
	verseMarkerRe    = regexp.MustCompile(`\\v\s+\d`)
	paraMarkerRe     = regexp.MustCompile(`^\\(p|m|q\d?|pi\d?|nb|b)\b`)
)

// normalizeUSFMChunk repairs the compact marker style found in package_version 3
// content (`\c1`, `\v1\v2`) into standard spaced USFM. Safe on well-formed input.
func normalizeUSFMChunk(text string) string {
	text = compactChapterRe.ReplaceAllString(text, `\c $1`)
	text = compactVerseRe.ReplaceAllString(text, `\v $1`)
	return strings.TrimSpace(text)
}

// convertBible assembles a single USFM file from ts chapter/chunk .txt files.
func convertBible(inDir, outDir string, m *ts.Manifest, rep *Report) error {
	book, ok := books.Lookup(m.ProjectID)
	if !ok {
		return fmt.Errorf("project id %q is not a valid book code", m.ProjectID)
	}

	title := readTextFile(filepath.Join(inDir, "front", "title.txt"))
	if title == "" {
		title = readTextFile(filepath.Join(inDir, "00", "title.txt"))
	}
	if title == "" {
		title = m.ProjectName
	}
	if title == "" {
		title = book.Title
		rep.Warnf("no book title found; using canonical %q", title)
	}

	code := strings.ToUpper(book.Code)
	var b strings.Builder
	fmt.Fprintf(&b, "\\id %s %s\n", code, title)
	b.WriteString("\\usfm 3.0\n\\ide UTF-8\n")
	fmt.Fprintf(&b, "\\h %s\n\\toc1 %s\n\\toc2 %s\n\\toc3 %s\n\\mt1 %s\n", title, title, title, title, title)

	dirs, err := listChunkDirs(inDir)
	if err != nil {
		return err
	}
	chunkCount := 0
	for _, d := range dirs {
		if d.Num == 0 { // 00/ holds only the book title
			continue
		}
		files, err := listChunkFiles(d.Path)
		if err != nil {
			rep.Warnf("chapter %s: %v", d.Name, err)
			continue
		}
		chapterNum := stripNum(d.Name)
		first := true
		for _, f := range files {
			if f.Num < 0 {
				continue // title.txt handled below, reference.txt not applicable
			}
			text := normalizeUSFMChunk(readTextFile(f.Path))
			if text == "" {
				rep.Warnf("chapter %s chunk %s: empty, skipped", d.Name, f.Base)
				continue
			}
			if first {
				if !chapterMarkerRe.MatchString(text) {
					b.WriteString("\\c " + chapterNum + "\n")
				}
				if cl := readTextFile(filepath.Join(d.Path, "title.txt")); cl != "" {
					b.WriteString("\\cl " + cl + "\n")
				}
				first = false
			}
			if !verseMarkerRe.MatchString(text) {
				rep.Warnf("chapter %s chunk %s: no \\v marker found; text included as-is", d.Name, f.Base)
			}
			if !paraMarkerRe.MatchString(text) {
				b.WriteString("\\p\n")
			}
			b.WriteString(text + "\n")
			chunkCount++
		}
	}
	if chunkCount == 0 {
		return fmt.Errorf("no content chunks found")
	}
	rep.Stats["chunks"] = chunkCount

	usfmName := fmt.Sprintf("%s-%s.usfm", book.USFMNum, code)
	if err := os.WriteFile(filepath.Join(outDir, usfmName), []byte(b.String()), 0o644); err != nil {
		return err
	}

	identifier := m.ResourceID
	if identifier == "" {
		identifier = "reg"
		rep.Warnf("empty resource id; defaulting identifier to %q", identifier)
	}
	rcTitle := m.ResourceName
	if rcTitle == "" {
		rcTitle = title
	}
	man := baseManifest(inDir, m, rep)
	man.DublinCore.Format = "text/usfm"
	man.DublinCore.Identifier = identifier
	man.DublinCore.Subject = "Bible"
	man.DublinCore.Title = rcTitle
	man.DublinCore.Type = "bundle"
	man.Projects = []rc.Project{{
		Title:         title,
		Versification: "ufw",
		Identifier:    book.Code,
		Sort:          book.Sort,
		Path:          "./" + usfmName,
		Categories:    []string{book.Category},
	}}
	return rc.WriteManifest(outDir, man)
}
