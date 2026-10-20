package localskills

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The declared settings are merged onto the shipped policy: lists added after
// the base's (each entry once), a set limit in place of the base's, and what a
// value leaves out said, never dropped silently.
func TestLoadPolicyMergesTheSettingsFile(t *testing.T) {
	home := t.TempDir()
	base := DefaultPolicy()
	cases := []struct {
		name  string
		file  string // "" = no settings file
		check func(t *testing.T, p Policy, notes []string)
		err   string
	}{
		{name: "no file is the shipped policy", check: func(t *testing.T, p Policy, notes []string) {
			if !slices.Equal(p.AgentDirs, base.AgentDirs) || !slices.Equal(p.Exclude, base.Exclude) || p.Limits != base.Limits || len(notes) != 0 {
				t.Errorf("got %+v %v", p, notes)
			}
		}},
		{name: "lines as arrays add after the base's, once", file: `{
			"extraSkillDirs": ["~/.mycli/skills", "~/.claude/skills", "` + filepath.Join(home, "tools", "skills") + `"],
			"excludes": ["*.log", ".git"],
			"secretNames": ["*.secret"],
			"secretPatterns": ["HOUSE-[0-9]{6}"],
			"maxSkills": "8"}`,
			check: func(t *testing.T, p Policy, notes []string) {
				want := append(slices.Clone(base.AgentDirs), "~/.mycli/skills", filepath.Join(home, "tools", "skills"))
				if !slices.Equal(p.AgentDirs, want) {
					t.Errorf("agent folders = %v, want %v", p.AgentDirs, want)
				}
				if !slices.Equal(p.Exclude, append(slices.Clone(base.Exclude), "*.log")) {
					t.Errorf("excludes = %v", p.Exclude)
				}
				if !slices.Contains(p.SecretNames, "*.secret") || len(p.SecretNames) != len(base.SecretNames)+1 {
					t.Errorf("secret names = %v", p.SecretNames)
				}
				if last := p.SecretRules[len(p.SecretRules)-1]; len(p.SecretRules) != len(base.SecretRules)+1 || last.Pattern != "HOUSE-[0-9]{6}" || last.Name != orgPatternName {
					t.Errorf("secret rules = %v", p.SecretRules)
				}
				if p.Limits.Skills != 8 || p.Limits.Files != base.Limits.Files {
					t.Errorf("limits = %+v", p.Limits)
				}
				if len(notes) != 0 {
					t.Errorf("notes = %v", notes)
				}
			}},
		{name: "a lines value as one string reads alike", file: `{"excludes": " *.log \n\n build "}`,
			check: func(t *testing.T, p Policy, _ []string) {
				if !slices.Equal(p.Exclude, append(slices.Clone(base.Exclude), "*.log", "build")) {
					t.Errorf("excludes = %v", p.Exclude)
				}
			}},
		{name: "a folder outside the home is left out with a note", file: `{"extraSkillDirs": ["/etc/skills", "relative/skills", "~/../other", "~", "~/.ok/skills"]}`,
			check: func(t *testing.T, p Policy, notes []string) {
				if !slices.Equal(p.AgentDirs, append(slices.Clone(base.AgentDirs), "~/.ok/skills")) {
					t.Errorf("agent folders = %v", p.AgentDirs)
				}
				if len(notes) != 4 || !strings.Contains(notes[0], "/etc/skills") || !strings.Contains(notes[0], "not a folder under your home") {
					t.Errorf("notes = %v", notes)
				}
			}},
		{name: "a pattern that does not compile is refused", file: `{"secretPatterns": ["ok", "("]}`, err: `pattern "("`},
		{name: "a skill count that is not a positive number is refused", file: `{"maxSkills": "0"}`, err: "maxSkills"},
		{name: "a skill count that is not a number is refused", file: `{"maxSkills": "many"}`, err: "maxSkills"},
		{name: "a malformed file is refused", file: `{"excludes": `, err: "not a JSON object"},
		{name: "a lines value of another shape is refused", file: `{"excludes": {"a": 1}}`, err: "excludes is not a list of lines"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := ""
			if c.file != "" {
				path = filepath.Join(t.TempDir(), "settings.json")
				write(t, path, c.file, 0o600)
			}
			p, notes, err := LoadPolicy(path, home)
			if c.err != "" {
				if err == nil || !strings.Contains(err.Error(), c.err) {
					t.Fatalf("err = %v, want one naming %q", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			c.check(t, p, notes)
		})
	}
	if _, _, err := LoadPolicy(filepath.Join(t.TempDir(), "missing.json"), home); err == nil {
		t.Error("a settings file that is named but missing is an error")
	}
}

// An organization's secret file name makes a skill ineligible, a member's
// exclude keeps a file out of the pack and the mirror, and a folder the
// settings left out is said in the state.
func TestServiceAppliesTheDeclaredSettings(t *testing.T) {
	s, home := newService(t)
	s.ComponentSettingsFile = filepath.Join(home, "component-settings.json")
	write(t, s.ComponentSettingsFile, `{"secretNames": ["*.secret"], "excludes": ["*.log"], "extraSkillDirs": ["/elsewhere/skills"]}`, 0o600)
	src := filepath.Join(home, "src")
	dir := skill(t, src, "alpha")
	write(t, filepath.Join(dir, "run.log"), "noise", 0o644)
	write(t, filepath.Join(dir, "notes.md"), "kept", 0o644)
	skill(t, src, "dirty")
	write(t, filepath.Join(src, "dirty", "db.secret"), "x", 0o644)

	st, err := s.Apply(Settings{Enabled: true, Sources: []string{src}, Skills: map[string]Choice{"alpha": {VM: true}, "dirty": {VM: true}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, sk := range st.Skills {
		if sk.Name == "dirty" && sk.Eligible {
			t.Errorf("a skill holding an organization's secret file name is not shared: %+v", sk)
		}
		if sk.Name == "alpha" && (!sk.Eligible || sk.Files != 2) {
			t.Errorf("an excluded file is not counted: %+v", sk)
		}
	}
	if !slices.ContainsFunc(st.Notes, func(n string) bool { return strings.Contains(n, "/elsewhere/skills") }) {
		t.Errorf("the folder the settings left out is said: %v", st.Notes)
	}
	archive, _, err := s.Pack("alpha")
	if err != nil {
		t.Fatal(err)
	}
	got, err := readTar(archive)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["alpha/run.log"]; ok || got["alpha/notes.md"] == nil {
		t.Errorf("the excluded file is not packed, the rest is: %v", got)
	}
	mirrored := filepath.Join(s.MirrorRoot, "alpha")
	if !exists(filepath.Join(mirrored, "notes.md")) || exists(filepath.Join(mirrored, "run.log")) {
		t.Errorf("the mirror holds the skill without the excluded file")
	}
	if exists(filepath.Join(s.MirrorRoot, "dirty")) {
		t.Error("an ineligible skill is never mirrored")
	}

	// A changed value applies at the next call, without a restart.
	write(t, s.ComponentSettingsFile, `{"secretPatterns": ["("]}`, 0o600)
	if _, err := s.State(); err == nil {
		t.Error("an organization pattern that does not compile is an error, not ignored")
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
