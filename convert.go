package ts2rc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/unfoldingWord/go-ts2rc/rc"
	"github.com/unfoldingWord/go-ts2rc/ts"
)

// baseManifest fills the dublin_core fields shared by every conversion class.
func baseManifest(inDir string, m *ts.Manifest, rep *Report) *rc.Manifest {
	date := rc.RepoDate(inDir)
	rights, fromFile := rc.DetectRights(inDir)
	if !fromFile {
		rep.Warnf("no license file found; defaulting rights to %q (tS standard)", rights)
	}
	langTitle := m.Language.Name
	if langTitle == "" {
		langTitle = m.Language.ID
	}
	contributors := m.Translators
	if len(contributors) == 0 {
		contributors = []string{"Door43 World Missions Community"}
		rep.Warnf("no translators in manifest; using default contributor")
	}
	var sources []rc.Source
	for _, s := range m.Sources {
		if s.Identifier == "" && s.Language == "" {
			continue
		}
		sources = append(sources, rc.Source{Identifier: s.Identifier, Language: s.Language, Version: s.Version})
	}
	return &rc.Manifest{
		DublinCore: rc.DublinCore{
			Conformsto:  "rc0.2",
			Contributor: contributors,
			Creator:     "Door43 World Missions Community",
			Description: fmt.Sprintf("Converted from translationStudio project (package_version %d, %s %s)", m.PackageVersion, m.GeneratorName, m.GeneratorBuild),
			Issued:      date,
			Language: rc.Language{
				Direction:  m.Language.Direction,
				Identifier: m.Language.ID,
				Title:      langTitle,
			},
			Modified:  date,
			Publisher: "Door43",
			Relation:  []string{},
			Rights:    rights,
			Source:    sources,
			Version:   "1",
		},
		Checking: rc.Checking{
			CheckingEntity: []string{"Door43 World Missions Community"},
			CheckingLevel:  "1",
		},
	}
}

// copyFirstOf copies the first existing candidate file from inDir to
// outDir/<destName>, reporting whether one was found.
func copyFirstOf(inDir, outDir, destName string, candidates []string) (bool, error) {
	for _, name := range candidates {
		src, err := os.Open(filepath.Join(inDir, name))
		if err != nil {
			continue
		}
		dst, err := os.Create(filepath.Join(outDir, destName))
		if err != nil {
			src.Close()
			return false, err
		}
		_, err = io.Copy(dst, src)
		src.Close()
		cerr := dst.Close()
		if err != nil {
			return false, err
		}
		if cerr != nil {
			return false, cerr
		}
		return true, nil
	}
	return false, nil
}

// copyLicense copies LICENSE.md into the RC repo when the ts repo has one.
func copyLicense(inDir, outDir string) (bool, error) {
	return copyFirstOf(inDir, outDir, "LICENSE.md", []string{"LICENSE.md", "LICENSE", "LICENSE.txt"})
}

// copyReadme carries a source README to the RC repo root when one exists.
// go-rc2sb then copies it on to the SB root (not as a metadata ingredient).
func copyReadme(inDir, outDir string) (bool, error) {
	return copyFirstOf(inDir, outDir, "README.md", []string{"README.md", "README", "README.txt", "readme.md"})
}

// Convert converts one translationStudio repository at inDir into an RC
// repository at outDir.
//
// It never panics. Failures are recorded on the returned Report (OK == false).
// Warnings are non-fatal. On hard conversion failure the incomplete outDir is removed.
//
// The resulting RC is suitable for conversion to Scripture Burrito via
// github.com/unfoldingWord/go-rc2sb.
func Convert(ctx context.Context, inDir, outDir string, opts Options) *Report {
	rep := &Report{Source: inDir, Output: outDir, Stats: map[string]int{}}

	fail := func(err error) *Report {
		rep.OK = false
		rep.Error = err.Error()
		rep.Err = err
		return rep
	}

	if err := ctx.Err(); err != nil {
		return fail(fmt.Errorf("context error: %w", err))
	}

	m, err := ts.LoadManifest(inDir)
	if err != nil {
		return fail(err)
	}
	rep.Version = m.PackageVersion
	if m.GeneratorName == "tc-desktop" {
		return fail(fmt.Errorf("manifest.json was written by tc-desktop: a translationCore project, not translationStudio (convert it with go-tc2rc)"))
	}
	rep.Class = m.Class()
	if rep.Class == "" {
		return fail(fmt.Errorf("unrecognized ts type %q (project %q)", m.TypeID, m.ProjectID))
	}
	if m.Language.ID == "" {
		return fail(fmt.Errorf("manifest has no target_language.id"))
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fail(err)
	}

	switch rep.Class {
	case "bible":
		err = convertBible(inDir, outDir, m, rep)
	case "obs":
		err = convertOBS(inDir, outDir, m, rep)
	case "tq", "tn", "obs-tq", "obs-tn":
		err = convertHelps(inDir, outDir, m, rep.Class, rep)
	case "tw":
		err = convertTW(inDir, outDir, m, opts, rep)
	case "ta":
		err = convertTA(inDir, outDir, m, rep)
	}
	if err != nil {
		os.RemoveAll(outDir) // don't leave a half-written RC repo behind
		return fail(err)
	}

	if err := ctx.Err(); err != nil {
		os.RemoveAll(outDir)
		return fail(fmt.Errorf("context error: %w", err))
	}

	copied, err := copyLicense(inDir, outDir)
	if err != nil {
		rep.Warnf("copying license: %v", err)
	}
	if !copied {
		if err := writeDefaultLicense(outDir); err != nil {
			rep.Warnf("writing default license: %v", err)
		} else {
			rep.Warnf("no license file in source; wrote default unfoldingWord CC BY-SA 4.0 LICENSE.md")
		}
	}
	if _, err := copyReadme(inDir, outDir); err != nil {
		rep.Warnf("copying README: %v", err)
	}
	rep.OK = true
	return rep
}

// ResultFromReport builds a Result summary from a completed Report.
func ResultFromReport(rep *Report) Result {
	if rep == nil {
		return Result{}
	}
	return Result{
		Class:    rep.Class,
		Version:  rep.Version,
		InDir:    rep.Source,
		OutDir:   rep.Output,
		Warnings: append([]string(nil), rep.Warnings...),
		Stats:    rep.Stats,
	}
}

// WriteReport writes the conversion report as JSON to path.
func WriteReport(rep *Report, path string) error {
	buf, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(buf, '\n'), 0o644)
}
