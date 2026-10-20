package localskills

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Other tools' skill folders are found under the home's hidden folders and
// the configuration and application support folders, a few levels deep, as
// paths under the home with a count and one name; known folders, links,
// excluded names and plain folders are not offered.
func TestFindCandidates(t *testing.T) {
	home := t.TempDir()
	skill(t, filepath.Join(home, ".newtool", "skills"), "lint")
	skill(t, filepath.Join(home, ".newtool", "skills"), "deploy")
	skill(t, filepath.Join(home, ".deep", "profiles", "work"), "notes") // a folder holding skills, not named skills
	skill(t, filepath.Join(home, ".config", "otheragent", "skills"), "review")
	skill(t, filepath.Join(home, "Library", "Application Support", "Editor", "skills"), "refactor")
	skill(t, filepath.Join(home, ".agents", "skills"), "known")                  // a default agent folder
	skill(t, filepath.Join(home, ".mine", "skills"), "source")                   // a source
	skill(t, filepath.Join(home, ".toodeep", "a", "b", "skills"), "far")         // past ScanDepth
	skill(t, filepath.Join(home, "visible", "skills"), "plain")                  // not hidden, not a root
	skill(t, filepath.Join(home, ".newtool", "node_modules", "pkg"), "excluded") // an excluded name
	write(t, filepath.Join(home, ".empty", "skills", "README.md"), "no skills", 0o644)
	if err := os.Symlink(filepath.Join(home, ".newtool"), filepath.Join(home, ".linked")); err != nil {
		t.Fatal(err)
	}

	got := FindCandidates(DefaultPolicy(), []string{"~/.mine/skills"}, home)
	want := []CandidateFolder{
		{Path: "~/.config/otheragent/skills", Skills: 1, Sample: "review"},
		{Path: "~/.deep/profiles/work", Skills: 1, Sample: "notes"},
		{Path: "~/.newtool/skills", Skills: 2, Sample: "deploy"},
		{Path: "~/Library/Application Support/Editor/skills", Skills: 1, Sample: "refactor"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("candidates = %+v\nwant %+v", got, want)
	}

	// An added extra agent folder is known: it is not offered again.
	p := DefaultPolicy().Merge(Policy{AgentDirs: []string{"~/.newtool/skills"}})
	for _, c := range FindCandidates(p, nil, home) {
		if c.Path == "~/.newtool/skills" {
			t.Fatal("an agent folder is offered again")
		}
	}
}

// The count of folders offered and the entries read in one tool's folder are
// bounded, and a big folder does not hide the next tool's.
func TestFindCandidatesIsBounded(t *testing.T) {
	home := t.TempDir()
	for i := range 30 {
		skill(t, filepath.Join(home, fmt.Sprintf(".tool%02d", i), "skills"), "s")
	}
	p := DefaultPolicy()
	if got := FindCandidates(p, nil, home); len(got) != p.Limits.Candidates {
		t.Fatalf("%d candidates, want the limit %d", len(got), p.Limits.Candidates)
	}
	home = t.TempDir()
	for i := range p.Limits.CandidateEntries + 10 {
		if err := os.MkdirAll(filepath.Join(home, ".aaacache", fmt.Sprintf("d%04d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	skill(t, filepath.Join(home, ".zzz", "skills"), "after")
	if got := FindCandidates(p, nil, home); len(got) != 1 || got[0].Path != "~/.zzz/skills" {
		t.Fatalf("after a big folder = %+v", got)
	}
}

// The state offers the candidates for the extra agent folders setting, and
// only names, never a file's content.
func TestStateOffersCandidates(t *testing.T) {
	s, home := newService(t)
	skill(t, filepath.Join(home, ".newtool", "skills"), "lint")
	st, err := s.State()
	if err != nil {
		t.Fatal(err)
	}
	if st.Candidates.Setting != SettingExtraSkillDirs || len(st.Candidates.Folders) != 1 || st.Candidates.Folders[0].Path != "~/.newtool/skills" {
		t.Fatalf("candidates = %+v", st.Candidates)
	}
	if strings.Contains(fmt.Sprint(st.Candidates), "description") {
		t.Fatal("a candidate carries a file's content")
	}
}
