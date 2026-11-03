package ts2rc

import "sort"

// projectSubjects maps the subject of each kind of translationStudio project Convert
// converts, named after the resource it translates (as DCS catalogs tS repos), to the
// conversion class that handles it.
var projectSubjects = map[string]string{
	"Bible":                     "bible",
	"Open Bible Stories":        "obs",
	"Translation Notes":         "tn",
	"Translation Questions":     "tq",
	"OBS Translation Notes":     "obs-tn",
	"OBS Translation Questions": "obs-tq",
	"Translation Words":         "tw",
	"Translation Academy":       "ta",
}

// Supports reports whether Convert converts a translationStudio project of the
// subject, named after the resource it translates (e.g. "Translation Notes").
func Supports(subject string) bool {
	_, ok := projectSubjects[subject]
	return ok
}

// SupportedSubjects returns the project subjects Convert converts, sorted.
func SupportedSubjects() []string {
	subjects := make([]string, 0, len(projectSubjects))
	for s := range projectSubjects {
		subjects = append(subjects, s)
	}
	sort.Strings(subjects)
	return subjects
}
