package ts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifestV7(t *testing.T) {
	dir := t.TempDir()
	manifest := `{
  "package_version": 7,
  "project": {"id": "gen", "name": "Genesis"},
  "type": {"id": "tn"},
  "resource": {"id": "tn", "name": "Translation Notes"},
  "target_language": {"id": "en", "name": "English", "direction": "ltr"},
  "source_translations": [
    {"resource_id": "ult", "language_id": "en", "version": "1"}
  ],
  "translators": ["Alice", "Bob"],
  "generator": {"name": "ts-desktop", "build": "200"}
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.PackageVersion != 7 || m.TypeID != "tn" || m.ProjectID != "gen" {
		t.Fatalf("unexpected manifest: %+v", m)
	}
	if m.Class() != "tn" {
		t.Fatalf("class: got %q", m.Class())
	}
	if len(m.Translators) != 2 || m.Language.ID != "en" {
		t.Fatalf("unexpected fields: %+v", m)
	}
	if len(m.Sources) != 1 || m.Sources[0].Identifier != "ult" {
		t.Fatalf("sources: %+v", m.Sources)
	}
}

func TestLoadManifestV3(t *testing.T) {
	dir := t.TempDir()
	manifest := `{
  "package_version": 3,
  "project": {"id": "act", "name": "Acts", "type": "text"},
  "resource_id": "udb",
  "target_language": {"id": "aa", "name": "Afar"},
  "source_translations": {
    "act-en-udb": {"version": "9"}
  },
  "translators": [{"name": "Translator One"}]
}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if m.PackageVersion != 3 || m.TypeID != "text" || m.ResourceID != "udb" {
		t.Fatalf("unexpected: %+v", m)
	}
	if m.Class() != "bible" {
		t.Fatalf("class: got %q", m.Class())
	}
	if len(m.Translators) != 1 || m.Translators[0] != "Translator One" {
		t.Fatalf("translators: %+v", m.Translators)
	}
	if len(m.Sources) != 1 || m.Sources[0].Language != "en" || m.Sources[0].Identifier != "udb" {
		t.Fatalf("sources: %+v", m.Sources)
	}
}

func TestClassOBS(t *testing.T) {
	m := &Manifest{TypeID: "text", ProjectID: "obs"}
	if m.Class() != "obs" {
		t.Fatalf("got %q", m.Class())
	}
	m = &Manifest{TypeID: "tq", ProjectID: "obs"}
	if m.Class() != "obs-tq" {
		t.Fatalf("got %q", m.Class())
	}
	m = &Manifest{TypeID: "tn", ProjectID: "obs"}
	if m.Class() != "obs-tn" {
		t.Fatalf("got %q", m.Class())
	}
}
