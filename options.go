// Package ts2rc converts translationStudio (ts) repositories to Resource Container (RC) format.
package ts2rc

import "fmt"

// Options configures a translationStudio to RC conversion.
type Options struct {
	// TWSourceDir is the path to a canonical Translation Words RC directory
	// (e.g. a local clone of unfoldingWord/en_tw). Used when converting tw
	// projects to place each word under bible/{kt|names|other}/ by matching
	// slug filenames. If empty, unmatched words fall back to bible/other/.
	TWSourceDir string
}

// Report is the fail-safe conversion record for one ts → RC conversion.
// Check OK to determine whether a usable RC repo was produced.
type Report struct {
	// Source is the input translationStudio directory.
	Source string `json:"source"`

	// Output is the output RC directory.
	Output string `json:"output"`

	// Class is the conversion class: bible, obs, tq, tn, obs-tq, obs-tn, or tw.
	Class string `json:"class"`

	// Version is the translationStudio package_version (3–8).
	Version int `json:"package_version"`

	// OK is true when a usable RC repository was written.
	OK bool `json:"ok"`

	// Error is set when OK is false.
	Error string `json:"error,omitempty"`

	// Err is the error behind Error, for callers that need to tell a failure
	// caused by the input (e.g. a manifest it can't read) from one caused by
	// the system (e.g. a full disk) with errors.Is and errors.As.
	Err error `json:"-"`

	// Warnings are non-fatal issues encountered during conversion.
	Warnings []string `json:"warnings,omitempty"`

	// Stats holds conversion counters (e.g. chunks, rows, stories, words).
	Stats map[string]int `json:"stats"`
}

// Warnf appends a formatted warning to the report.
func (r *Report) Warnf(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// Result holds summary information about a completed conversion.
// It mirrors go-rc2sb's Result shape for library consumers.
type Result struct {
	// Class is the conversion class (bible, obs, tq, tn, obs-tq, obs-tn, tw).
	Class string

	// Version is the translationStudio package_version.
	Version int

	// InDir is the input translationStudio directory.
	InDir string

	// OutDir is the output RC directory.
	OutDir string

	// Warnings is the list of non-fatal conversion warnings.
	Warnings []string

	// Stats holds conversion counters.
	Stats map[string]int
}
