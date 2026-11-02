// Package ts parses and normalizes translationStudio manifest.json files.
package ts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Manifest is the normalized view of a translationStudio manifest.json.
// It is parsed from a map so that the schemas (package_version 1–2 ts-android
// slugs, 3, and 5–8) and all their lenient value typings (int/string/junk)
// normalize without error.
type Manifest struct {
	PackageVersion int
	TypeID         string // text | tq | tn | tw | ta
	ProjectID      string // book code | obs | bible (tw) | vol1/vol2 (ta)
	ProjectName    string
	ResourceID     string
	ResourceName   string
	Format         string
	GeneratorName  string
	GeneratorBuild string
	Language       Language
	Sources        []Source
	Translators    []string
}

// Language is the target language of a ts project.
type Language struct {
	ID        string
	Name      string
	Direction string
}

// Source is one entry from source_translations.
type Source struct {
	Identifier string // resource id, e.g. ulb, obs
	Language   string
	Version    string
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

func asInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	default:
		return 0
	}
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return map[string]any{}
}

// LoadManifest reads and normalizes <repoDir>/manifest.json.
func LoadManifest(repoDir string) (*Manifest, error) {
	buf, err := os.ReadFile(filepath.Join(repoDir, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("reading manifest.json: %w", err)
	}
	buf = []byte(strings.TrimPrefix(string(buf), "\uFEFF"))
	var raw map[string]any
	if err := json.Unmarshal(buf, &raw); err != nil {
		return nil, fmt.Errorf("parsing manifest.json: %w", err)
	}

	m := &Manifest{}
	m.PackageVersion = asInt(raw["package_version"])
	if m.PackageVersion < 1 {
		return nil, fmt.Errorf("package_version %d is not a tS manifest (want 1-8)", m.PackageVersion)
	}

	// project: project.id, else v3 flat project_id, else v1–2 slug
	proj := getMap(raw, "project")
	m.ProjectID = strings.ToLower(strings.TrimSpace(asString(proj["id"])))
	if m.ProjectID == "" {
		m.ProjectID = strings.ToLower(strings.TrimSpace(asString(raw["project_id"])))
	}
	if m.ProjectID == "" {
		m.ProjectID = strings.ToLower(strings.TrimSpace(asString(raw["slug"])))
	}
	m.ProjectName = strings.TrimSpace(asString(proj["name"]))

	// type: v5+ type.id, v3 project.type; default "text"
	m.TypeID = strings.ToLower(strings.TrimSpace(asString(getMap(raw, "type")["id"])))
	if m.TypeID == "" {
		m.TypeID = strings.ToLower(strings.TrimSpace(asString(proj["type"])))
	}
	if m.TypeID == "" {
		m.TypeID = "text"
	}

	// resource: v5+ resource.id/.name, v3 flat resource_id
	res := getMap(raw, "resource")
	m.ResourceID = strings.ToLower(strings.TrimSpace(asString(res["id"])))
	m.ResourceName = strings.TrimSpace(asString(res["name"]))
	if m.ResourceID == "" {
		m.ResourceID = strings.ToLower(strings.TrimSpace(asString(raw["resource_id"])))
	}

	m.Format = strings.ToLower(strings.TrimSpace(asString(raw["format"])))
	gen := getMap(raw, "generator")
	m.GeneratorName = asString(gen["name"])
	m.GeneratorBuild = asString(gen["build"])

	// target_language: id, or v1–2 slug
	tl := getMap(raw, "target_language")
	langID := strings.TrimSpace(asString(tl["id"]))
	if langID == "" {
		langID = strings.TrimSpace(asString(tl["slug"]))
	}
	m.Language = Language{
		ID:        langID,
		Name:      strings.TrimSpace(asString(tl["name"])),
		Direction: strings.TrimSpace(asString(tl["direction"])),
	}
	if m.Language.Direction == "" {
		m.Language.Direction = "ltr"
	}

	// source_translations: v3 object keyed "{proj}-{lang}-{res}", v5+ array
	switch st := raw["source_translations"].(type) {
	case []any:
		for _, e := range st {
			em, ok := e.(map[string]any)
			if !ok {
				continue
			}
			m.Sources = append(m.Sources, Source{
				Identifier: strings.ToLower(asString(em["resource_id"])),
				Language:   asString(em["language_id"]),
				Version:    asString(em["version"]),
			})
		}
	case map[string]any:
		for key, e := range st {
			em, _ := e.(map[string]any)
			src := Source{Version: asString(em["version"])}
			// key form: {project}-{lang}-{resource}
			parts := strings.Split(key, "-")
			if len(parts) >= 3 {
				src.Language = strings.Join(parts[1:len(parts)-1], "-")
				src.Identifier = strings.ToLower(parts[len(parts)-1])
			}
			m.Sources = append(m.Sources, src)
		}
	}

	// translators: v3 array of {name}, v5+ array of strings
	if translators, ok := raw["translators"].([]any); ok {
		for _, e := range translators {
			switch t := e.(type) {
			case string:
				if s := strings.TrimSpace(t); s != "" {
					m.Translators = append(m.Translators, s)
				}
			case map[string]any:
				if s := strings.TrimSpace(asString(t["name"])); s != "" {
					m.Translators = append(m.Translators, s)
				}
			}
		}
	}

	return m, nil
}

// Class returns the conversion class for dispatch.
// One of: bible, obs, tq, tn, obs-tq, obs-tn, tw, ta — or "" if unrecognized.
// A text project named tw or ta is a help with no type recorded, not a book.
func (m *Manifest) Class() string {
	switch m.TypeID {
	case "text":
		switch m.ProjectID {
		case "obs", "tw", "ta":
			return m.ProjectID
		}
		return "bible"
	case "tq":
		if m.ProjectID == "obs" {
			return "obs-tq"
		}
		return "tq"
	case "tn":
		if m.ProjectID == "obs" {
			return "obs-tn"
		}
		return "tn"
	case "tw":
		return "tw"
	case "ta":
		return "ta"
	}
	return ""
}
