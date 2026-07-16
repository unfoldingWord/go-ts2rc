package ts2rc

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConvertMissingManifest(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(t.TempDir(), "out")
	rep := Convert(context.Background(), dir, out, Options{})
	if rep.OK {
		t.Fatal("expected failure for missing manifest")
	}
	if rep.Error == "" {
		t.Fatal("expected error message")
	}
}

func TestConvertCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	rep := Convert(ctx, t.TempDir(), filepath.Join(t.TempDir(), "out"), Options{})
	if rep.OK {
		t.Fatal("expected failure for cancelled context")
	}
}

func TestConvertUnrecognizedType(t *testing.T) {
	dir := t.TempDir()
	manifest := `{
  "package_version": 7,
  "project": {"id": "gen", "name": "Genesis"},
  "type": {"id": "unknown"},
  "target_language": {"id": "en", "name": "English", "direction": "ltr"}
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := Convert(context.Background(), dir, filepath.Join(t.TempDir(), "out"), Options{})
	if rep.OK {
		t.Fatal("expected failure for unrecognized type")
	}
}

func TestConvertMissingLanguage(t *testing.T) {
	dir := t.TempDir()
	manifest := `{
  "package_version": 7,
  "project": {"id": "gen", "name": "Genesis"},
  "type": {"id": "text"},
  "target_language": {"id": "", "name": "English"}
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	rep := Convert(context.Background(), dir, filepath.Join(t.TempDir(), "out"), Options{})
	if rep.OK {
		t.Fatal("expected failure for missing language id")
	}
}
