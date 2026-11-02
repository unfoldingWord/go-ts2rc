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

func writeMinimalBible(t *testing.T, dir string) {
	t.Helper()
	manifest := `{
  "package_version": 7,
  "project": {"id": "tit", "name": "Tito"},
  "type": {"id": "text"},
  "resource": {"id": "reg", "name": "Regular"},
  "target_language": {"id": "es-419", "name": "Español Latin America", "direction": "ltr"},
  "translators": ["Test Translator"]
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	writeChunk := func(rel, content string) {
		t.Helper()
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeChunk("front/title.txt", "Tito")
	writeChunk("01/title.txt", "Capítulo 1")
	writeChunk("01/01.txt", `\c 1 \v 1 Pablo, un siervo de Dios \v 2 con la esperanza segura`)
	writeChunk("01/03.txt", `\v 3 Pero en el tiempo correcto \v 4 A Tito`)
	writeChunk("03/01.txt", `\c 3 \v 1 Recuerdales someterse`)
}

func TestConvertBibleUSFMLayout(t *testing.T) {
	inDir := t.TempDir()
	writeMinimalBible(t, inDir)
	outDir := filepath.Join(t.TempDir(), "rc")

	rep := Convert(context.Background(), inDir, outDir, Options{})
	if !rep.OK {
		t.Fatalf("conversion failed: %s", rep.Error)
	}
	if rep.Class != "bible" {
		t.Fatalf("class: got %q", rep.Class)
	}

	buf, err := os.ReadFile(filepath.Join(outDir, "57-TIT.usfm"))
	if err != nil {
		t.Fatal(err)
	}

	want := `\id TIT Tito
\usfm 3.0
\ide UTF-8
\h Tito
\toc1 Tito
\toc2 Tito
\toc3 Tito
\mt1 Tito

\c 1
\cl Capítulo 1
\p
\v 1 Pablo, un siervo de Dios
\v 2 con la esperanza segura
\p
\v 3 Pero en el tiempo correcto
\v 4 A Tito

\c 3
\p
\v 1 Recuerdales someterse
`
	if got := string(buf); got != want {
		t.Fatalf("USFM layout mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestFormatChunkUSFM(t *testing.T) {
	in := `\v 4 A Tito, un verdadero hijo. \p \v 5 Para este propósito \q1 linea poetica`
	want := "\\v 4 A Tito, un verdadero hijo.\n\\p\n\\v 5 Para este propósito\n\\q1 linea poetica"
	if got := formatChunkUSFM(in); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	// package_version 3 compact style is normalized before formatting
	if got := formatChunkUSFM(normalizeUSFMChunk(`\v1 uno \v2 dos`)); got != "\\v 1 uno\n\\v 2 dos" {
		t.Fatalf("compact: got:\n%s", got)
	}
	// already-formatted input is unchanged (idempotent)
	formatted := "\\p\n\\v 1 uno\n\\v 2 dos"
	if got := formatChunkUSFM(formatted); got != formatted {
		t.Fatalf("idempotency: got:\n%s", got)
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

func TestConvertTA(t *testing.T) {
	inDir := t.TempDir()
	manifest := `{"package_version": 6, "project": {"id": "vol1", "name": "translationAcademy Vol 1"}, "type": {"id": "ta"}, "target_language": {"id": "fr", "name": "français"}}`
	if err := os.WriteFile(filepath.Join(inDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := filepath.Join(inDir, "01")
	if err := os.MkdirAll(ch, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, chunk := range map[string]string{
		"vol1_translate_figs_metaphor.txt": `[{"title":"Métaphore","body":"Une métaphore est..."}]`,
		"vol1_intro_ta_intro.txt":          `[{"title":"Introduction","body":"Bienvenue"}]`,
	} {
		if err := os.WriteFile(filepath.Join(ch, name), []byte(chunk), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	outDir := filepath.Join(t.TempDir(), "rc")

	rep := Convert(context.Background(), inDir, outDir, Options{})
	if !rep.OK {
		t.Fatalf("conversion failed: %s", rep.Error)
	}
	if rep.Class != "ta" || rep.Stats["articles"] != 2 {
		t.Fatalf("class %q stats %v", rep.Class, rep.Stats)
	}
	if got := readTextFile(filepath.Join(outDir, "translate", "figs_metaphor", "title.md")); got != "Métaphore" {
		t.Fatalf("title.md: %q", got)
	}
	if got := readTextFile(filepath.Join(outDir, "intro", "ta_intro", "01.md")); got != "Bienvenue" {
		t.Fatalf("01.md: %q", got)
	}
	s := readTextFile(filepath.Join(outDir, "manifest.yaml"))
	intro, translate := strings.Index(s, "identifier: intro"), strings.Index(s, "identifier: translate")
	if !strings.Contains(s, "subject: Translation Academy") || intro < 0 || translate < intro {
		t.Fatalf("manifest should list intro before translate under Translation Academy:\n%s", s)
	}
}

func TestConvertV2Bible(t *testing.T) {
	inDir := t.TempDir()
	manifest := `{"package_version": 2, "slug": "phm", "target_language": {"slug": "aa", "name": "Afaraf", "direction": "ltr"}, "generator": {"name": "ts-android", "build": 104}}`
	if err := os.WriteFile(filepath.Join(inDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	ch := filepath.Join(inDir, "01")
	if err := os.MkdirAll(ch, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ch, "01.txt"), []byte("\\c 1\n\\p\n\\v 1 Grace be with you."), 0o644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(t.TempDir(), "rc")

	rep := Convert(context.Background(), inDir, outDir, Options{})
	if !rep.OK {
		t.Fatalf("conversion failed: %s", rep.Error)
	}
	if !strings.Contains(readTextFile(filepath.Join(outDir, "58-PHM.usfm")), `\v 1 Grace be with you.`) {
		t.Fatal("58-PHM.usfm is missing the verse")
	}
}

func TestConvertRejectsTCDesktopManifest(t *testing.T) {
	inDir := t.TempDir()
	manifest := `{"package_version": 1, "generator": {"name": "tc-desktop"}, "project": {"id": "tit"}, "target_language": {"id": "abu"}}`
	if err := os.WriteFile(filepath.Join(inDir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := Convert(context.Background(), inDir, filepath.Join(t.TempDir(), "rc"), Options{})
	if rep.OK || !strings.Contains(rep.Error, "translationCore") {
		t.Fatalf("want a translationCore error, got OK=%v %q", rep.OK, rep.Error)
	}
}
