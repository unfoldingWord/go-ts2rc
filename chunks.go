package ts2rc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// chunkDir is a numeric content directory (chapter, story, or tw group).
type chunkDir struct {
	Name string // as on disk, e.g. "01"
	Num  int
	Path string
}

// chunkFile is a file inside a chunk dir.
type chunkFile struct {
	Base string // filename without .txt, e.g. "01", "title", "aaron"
	Num  int    // numeric value, -1 for non-numeric names
	Path string
}

var numericRe = regexp.MustCompile(`^\d+$`)

// listChunkDirs returns the numeric directories of a ts repo, sorted numerically.
func listChunkDirs(repoDir string) ([]chunkDir, error) {
	entries, err := os.ReadDir(repoDir)
	if err != nil {
		return nil, err
	}
	var dirs []chunkDir
	for _, e := range entries {
		if !e.IsDir() || !numericRe.MatchString(e.Name()) {
			continue
		}
		n, _ := strconv.Atoi(e.Name())
		dirs = append(dirs, chunkDir{Name: e.Name(), Num: n, Path: filepath.Join(repoDir, e.Name())})
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Num < dirs[j].Num })
	return dirs, nil
}

// listChunkFiles returns .txt files in a chunk dir; numeric ones sorted
// numerically first, then non-numeric (title, reference, word slugs) alphabetically.
func listChunkFiles(dir string) ([]chunkFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []chunkFile
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".txt")
		num := -1
		if numericRe.MatchString(base) {
			num, _ = strconv.Atoi(base)
		}
		files = append(files, chunkFile{Base: base, Num: num, Path: filepath.Join(dir, e.Name())})
	}
	sort.Slice(files, func(i, j int) bool {
		a, b := files[i], files[j]
		if (a.Num >= 0) != (b.Num >= 0) {
			return a.Num >= 0 // numeric first
		}
		if a.Num >= 0 {
			return a.Num < b.Num
		}
		return a.Base < b.Base
	})
	return files, nil
}

// readTextFile reads a file, trims it, and strips a UTF-8 BOM.
func readTextFile(path string) string {
	buf, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(string(buf), "\uFEFF"))
}

// TitleBody is one entry of a ts tq/tn/tw JSON chunk.
type TitleBody struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// readTitleBodyChunk parses a ts help chunk file (JSON array of {title, body}).
func readTitleBodyChunk(path string) ([]TitleBody, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(strings.TrimPrefix(string(buf), "\uFEFF"))
	var entries []TitleBody
	if err := json.Unmarshal([]byte(s), &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// stripNum removes leading zeros: "01" -> "1"; empty stays empty.
func stripNum(s string) string {
	t := strings.TrimLeft(s, "0")
	if t == "" && s != "" {
		return "0"
	}
	return t
}
