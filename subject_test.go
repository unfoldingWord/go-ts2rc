package ts2rc

import (
	"os"
	"regexp"
	"sort"
	"testing"
)

func TestSupports(t *testing.T) {
	if !Supports("Translation Notes") || !Supports("Translation Academy") || Supports("Greek Lexicon") || Supports("TSV Translation Notes") {
		t.Error("Supports does not match the project subjects")
	}
	if got := SupportedSubjects(); len(got) != len(projectSubjects) || !sort.StringsAreSorted(got) {
		t.Errorf("SupportedSubjects() = %q", got)
	}
}

// TestProjectSubjectsCoverConvertClasses keeps projectSubjects in step with the
// classes Convert dispatches on.
func TestProjectSubjectsCoverConvertClasses(t *testing.T) {
	src, err := os.ReadFile("convert.go")
	if err != nil {
		t.Fatal(err)
	}
	sw := regexp.MustCompile(`(?s)switch rep\.Class \{(.*?)\n\t\}`).FindSubmatch(src)
	if sw == nil {
		t.Fatal("no switch on rep.Class in convert.go")
	}
	classes := map[string]bool{}
	for _, m := range regexp.MustCompile(`"([a-z-]+)"`).FindAllSubmatch(sw[1], -1) {
		classes[string(m[1])] = true
	}
	mapped := map[string]bool{}
	for _, class := range projectSubjects {
		mapped[class] = true
	}
	for class := range classes {
		if !mapped[class] {
			t.Errorf("Convert handles class %q, but no project subject maps to it", class)
		}
	}
	for class := range mapped {
		if !classes[class] {
			t.Errorf("project subject maps to class %q, which Convert does not handle", class)
		}
	}
}
