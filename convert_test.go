package ts2rc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMinimalTN(t *testing.T, dir string) {
	t.Helper()
	manifest := `{
  "package_version": 7,
  "project": {"id": "gen", "name": "Genesis"},
  "type": {"id": "tn"},
  "resource": {"id": "tn", "name": "Translation Notes"},
  "target_language": {"id": "en", "name": "English", "direction": "ltr"},
  "translators": ["Test Translator"]
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := filepath.Join(dir, "01")
	if err := os.MkdirAll(ch, 0o755); err != nil {
		t.Fatal(err)
	}
	chunk := `[{"title":"in the beginning","body":"This means when God started creating."}]`
	if err := os.WriteFile(filepath.Join(ch, "01.txt"), []byte(chunk), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestConvertTN(t *testing.T) {
	inDir := t.TempDir()
	writeMinimalTN(t, inDir)
	outDir := filepath.Join(t.TempDir(), "rc")

	rep := Convert(context.Background(), inDir, outDir, Options{})
	if !rep.OK {
		t.Fatalf("conversion failed: %s", rep.Error)
	}
	if rep.Class != "tn" {
		t.Fatalf("class: got %q", rep.Class)
	}
	if rep.Stats["rows"] != 1 {
		t.Fatalf("rows: got %v", rep.Stats)
	}

	// RC artifacts
	if _, err := os.Stat(filepath.Join(outDir, "manifest.yaml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "tn_GEN.tsv")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "LICENSE.md")); err != nil {
		t.Fatal(err)
	}

	buf, err := os.ReadFile(filepath.Join(outDir, "manifest.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(buf)
	for _, want := range []string{"TSV Translation Notes", "subject:", "tn", "text/tsv"} {
		if !strings.Contains(s, want) {
			t.Fatalf("manifest missing %q:\n%s", want, s)
		}
	}
}

func TestWriteReport(t *testing.T) {
	rep := &Report{Source: "a", Output: "b", OK: true, Stats: map[string]int{"rows": 1}}
	path := filepath.Join(t.TempDir(), "r.json")
	if err := WriteReport(rep, path); err != nil {
		t.Fatal(err)
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(buf), `"ok": true`) {
		t.Fatalf("unexpected report: %s", buf)
	}
}
