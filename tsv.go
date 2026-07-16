package ts2rc

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"

	"github.com/unfoldingWord/go-ts2rc/books"
	"github.com/unfoldingWord/go-ts2rc/rc"
	"github.com/unfoldingWord/go-ts2rc/ts"
)

const idChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// rowID makes a deterministic 4-char TSV row ID (letter + 3 alphanumerics),
// unique within one file.
func rowID(seed string, seen map[string]bool) string {
	h := fnv.New32a()
	h.Write([]byte(seed))
	for {
		v := h.Sum32()
		id := string(idChars[v%26]) + // first char must be a letter
			string(idChars[(v/26)%36]) +
			string(idChars[(v/26/36)%36]) +
			string(idChars[(v/26/36/36)%36])
		if !seen[id] {
			seen[id] = true
			return id
		}
		h.Write([]byte("x"))
	}
}

// tsvEscape cleans a value for a TSV cell: tabs become spaces, newlines become
// the literal two-character sequence \n (the convention used by en_tn etc).
func tsvEscape(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	return strings.ReplaceAll(s, "\n", "\\n")
}

// convertHelps converts ts tq/tn JSON chunks (book or OBS) into a single RC TSV.
// class is one of: tq, tn, obs-tq, obs-tn.
func convertHelps(inDir, outDir string, m *ts.Manifest, class string, rep *Report) error {
	isOBS := strings.HasPrefix(class, "obs-")
	isTN := strings.HasSuffix(class, "tn")

	var book books.BookInfo
	if isOBS {
		book = books.OBS
	} else {
		var ok bool
		book, ok = books.Lookup(m.ProjectID)
		if !ok {
			return fmt.Errorf("project id %q is not a valid book code", m.ProjectID)
		}
	}

	var header []string
	if isTN {
		header = []string{"Reference", "ID", "Tags", "SupportReference", "Quote", "Occurrence", "Note"}
	} else {
		header = []string{"Reference", "ID", "Tags", "Quote", "Occurrence", "Question", "Response"}
	}

	var rows [][]string
	seen := map[string]bool{}
	dirs, err := listChunkDirs(inDir)
	if err != nil {
		return err
	}
	for _, d := range dirs {
		files, err := listChunkFiles(d.Path)
		if err != nil {
			rep.Warnf("chapter %s: %v", d.Name, err)
			continue
		}
		ch := stripNum(d.Name)
		for _, f := range files {
			verse := stripNum(f.Base)
			tags := ""
			if f.Num < 0 {
				if f.Base == "title" { // story/chapter title notes (OBS convention: ref N:0, tag "title")
					verse, tags = "0", "title"
				} else if f.Base == "reference" {
					continue
				} else {
					rep.Warnf("chapter %s: unexpected chunk file %q skipped", d.Name, f.Base)
					continue
				}
			}
			entries, err := readTitleBodyChunk(f.Path)
			if err != nil {
				rep.Warnf("chapter %s chunk %s: not a valid JSON chunk (%v), skipped", d.Name, f.Base, err)
				continue
			}
			ref := ch + ":" + verse
			for i, e := range entries {
				title, body := tsvEscape(e.Title), tsvEscape(e.Body)
				if title == "" && body == "" {
					continue
				}
				id := rowID(fmt.Sprintf("%s|%d|%s|%s", ref, i, title, body), seen)
				if isTN {
					rows = append(rows, []string{ref, id, tags, "", title, "1", body})
				} else {
					occ := ""
					if !isOBS {
						occ = "0"
					}
					rows = append(rows, []string{ref, id, tags, "", occ, title, body})
				}
			}
		}
	}
	if len(rows) == 0 {
		return fmt.Errorf("no %s entries found", class)
	}
	rep.Stats["rows"] = len(rows)

	prefix := "tq_"
	if isTN {
		prefix = "tn_"
	}
	fileName := prefix + strings.ToUpper(book.Code) + ".tsv"
	var b strings.Builder
	b.WriteString(strings.Join(header, "\t") + "\n")
	for _, r := range rows {
		b.WriteString(strings.Join(r, "\t") + "\n")
	}
	if err := os.WriteFile(filepath.Join(outDir, fileName), []byte(b.String()), 0o644); err != nil {
		return err
	}

	subjects := map[string]string{
		"tq":     "TSV Translation Questions",
		"tn":     "TSV Translation Notes",
		"obs-tq": "TSV OBS Translation Questions",
		"obs-tn": "TSV OBS Translation Notes",
	}
	titles := map[string]string{
		"tq":     "Translation Questions",
		"tn":     "Translation Notes",
		"obs-tq": "Open Bible Stories Translation Questions",
		"obs-tn": "Open Bible Stories Translation Notes",
	}

	man := baseManifest(inDir, m, rep)
	man.DublinCore.Format = "text/tsv"
	man.DublinCore.Identifier = class
	man.DublinCore.Subject = subjects[class]
	man.DublinCore.Title = titles[class]
	man.DublinCore.Type = "help"
	if isOBS {
		man.DublinCore.Relation = []string{m.Language.ID + "/obs"}
	}
	proj := rc.Project{
		Title:         titles[class],
		Versification: "ufw",
		Identifier:    book.Code,
		Sort:          book.Sort,
		Path:          "./" + fileName,
	}
	if !isOBS {
		proj.Title = book.Title
		proj.Categories = []string{book.Category}
	}
	man.Projects = []rc.Project{proj}
	return rc.WriteManifest(outDir, man)
}
