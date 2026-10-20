package localskills

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CandidateRoot is a folder whose children may be tools that keep skills (the
// home's dot-folders, an application support folder). Hidden takes only its
// hidden children (a name starting with ".").
type CandidateRoot struct {
	Path   string `json:"path"`
	Hidden bool   `json:"hidden,omitempty"`
}

// Candidates are folders on the member's computer where another tool keeps
// skills, found by the component for the member to add to Setting (the
// declared setting that takes them: extraSkillDirs). Only names and counts:
// a candidate is a folder whose sub-folders hold a SKILL.md, and no file is
// ever read.
type Candidates struct {
	Setting string            `json:"setting"`
	Folders []CandidateFolder `json:"folders"`
}

// CandidateFolder is one folder that holds skills: its path under the home
// ("~/.tool/skills"), how many skills it holds and the first one's name.
type CandidateFolder struct {
	Path   string `json:"path"`
	Skills int    `json:"skills"`
	Sample string `json:"sample"`
}

// FindCandidates looks, in each candidate root of the policy, at each tool
// folder and up to ScanDepth levels inside it for a folder holding skills
// (sub-folders with a SKILL.md). It reads at most CandidateEntries entries
// of one tool folder (a big cache folder cannot starve the others) and
// answers at most Limits.Candidates folders, never one that is not under
// home, a link, excluded by name, or already known (a source or an agent
// folder of the policy, or inside a source).
func FindCandidates(p Policy, sources []string, home string) []CandidateFolder {
	known := map[string]bool{}
	for _, d := range p.AgentDirs {
		known[filepath.Clean(ExpandHome(d, home))] = true
	}
	var srcs []string
	for _, s := range sources {
		srcs = append(srcs, filepath.Clean(ExpandHome(s, home)))
	}
	skip := func(dir string) bool {
		if known[dir] {
			return true
		}
		for _, s := range srcs {
			if within(s, dir) {
				return true
			}
		}
		return false
	}
	var out []CandidateFolder
	entries := 0 // read in the tool folder being looked at
	read := func(dir string, limit int) []os.DirEntry {
		list, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		if room := limit - entries; len(list) > room {
			list = list[:max(room, 0)]
		}
		entries += len(list)
		return list
	}
	full := func() bool { return len(out) >= p.Limits.Candidates || entries >= p.Limits.CandidateEntries }
	var visit func(dir string, depth int)
	visit = func(dir string, depth int) {
		if full() {
			return
		}
		var dirs []string
		skills := 0
		sample := ""
		for _, e := range read(dir, p.Limits.CandidateEntries) {
			child := filepath.Join(dir, e.Name())
			if matchAny(p.Exclude, e.Name()) {
				continue
			}
			if st, err := os.Stat(child); err != nil || !st.IsDir() {
				continue
			}
			if _, err := os.Stat(filepath.Join(child, "SKILL.md")); err == nil {
				skills++
				if sample == "" || e.Name() < sample {
					sample = e.Name()
				}
				continue
			}
			if e.Type()&os.ModeSymlink == 0 {
				dirs = append(dirs, child)
			}
		}
		if skills > 0 && !skip(dir) {
			if rel, err := filepath.Rel(home, dir); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				out = append(out, CandidateFolder{Path: "~/" + filepath.ToSlash(rel), Skills: skills, Sample: sample})
			}
		}
		if depth >= p.Limits.ScanDepth {
			return
		}
		for _, d := range dirs {
			if full() {
				return
			}
			visit(d, depth+1)
		}
	}
	for _, root := range p.CandidateRoots {
		base := filepath.Clean(ExpandHome(root.Path, home))
		if !within(home, base) {
			continue
		}
		list, _ := os.ReadDir(base)
		for _, e := range list {
			if len(out) >= p.Limits.Candidates {
				break
			}
			name := e.Name()
			if (root.Hidden && !strings.HasPrefix(name, ".")) || !e.IsDir() || matchAny(p.Exclude, name) {
				continue
			}
			tool := filepath.Join(base, name)
			if isRoot(p.CandidateRoots, tool, home) {
				continue // looked at as a root of its own
			}
			entries = 0
			visit(tool, 0)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// isRoot reports a folder that is itself a candidate root (~/.config under ~).
func isRoot(roots []CandidateRoot, dir, home string) bool {
	for _, r := range roots {
		if filepath.Clean(ExpandHome(r.Path, home)) == dir {
			return true
		}
	}
	return false
}
