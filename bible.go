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
	verseMarkerRe    = regexp.MustCompile(`\\v\s+\d`)
	paraMarkerRe     = regexp.MustCompile(`^\\(pi\d?|q\d?|nb|p|m|b)\b`)
	leadingChapterRe = regexp.MustCompile(`^\\c\s+(\d+)\s*`)
	verseBreakRe     = regexp.MustCompile(`[ \t]*(\\v\s+\d)`)
	paraBreakRe      = regexp.MustCompile(`[ \t]*(\\(?:pi\d?|q\d?|nb|p|m|b)\b)`)
)

// normalizeUSFMChunk repairs the compact marker style found in package_version 3
// content (`\c1`, `\v1\v2`) into standard spaced USFM. Safe on well-formed input.
func normalizeUSFMChunk(text string) string {
	text = compactChapterRe.ReplaceAllString(text, `\c $1`)
	text = compactVerseRe.ReplaceAllString(text, `\v $1`)
	return strings.TrimSpace(text)
}

// formatChunkUSFM puts every verse and paragraph-level marker of a chunk on
// its own line and drops blank lines, so assembled chapters read one marker
// per line instead of the single run-on line found in ts chunk files.
func formatChunkUSFM(text string) string {
	text = verseBreakRe.ReplaceAllString(text, "\n$1")
	text = paraBreakRe.ReplaceAllString(text, "\n$1")
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
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

		// Format all chunks first so the chapter heading is only written when
		// the chapter actually has content.
		chapterNum := stripNum(d.Name)
		var chunks []string
		for _, f := range files {
			if f.Num < 0 {
				continue // title.txt handled below, reference.txt not applicable
			}
			text := normalizeUSFMChunk(readTextFile(f.Path))
			if text == "" {
				rep.Warnf("chapter %s chunk %s: empty, skipped", d.Name, f.Base)
				continue
			}
			// Relocate a leading \c marker: it is re-emitted on its own line
			// at the top of the chapter, above the first \p.
			if m := leadingChapterRe.FindStringSubmatch(text); m != nil {
				chapterNum = m[1]
				text = strings.TrimSpace(text[len(m[0]):])
				if text == "" {
					continue // chunk held only the \c marker
				}
			}
			if !verseMarkerRe.MatchString(text) {
				rep.Warnf("chapter %s chunk %s: no \\v marker found; text included as-is", d.Name, f.Base)
			}
			chunks = append(chunks, formatChunkUSFM(text))
		}
		if len(chunks) == 0 {
			continue
		}

		// Blank line between the header/previous chapter and this chapter.
		b.WriteString("\n")
		b.WriteString("\\c " + chapterNum + "\n")
		if cl := readTextFile(filepath.Join(d.Path, "title.txt")); cl != "" {
			b.WriteString("\\cl " + cl + "\n")
		}
		for _, text := range chunks {
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
