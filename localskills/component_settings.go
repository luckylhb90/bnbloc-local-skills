package localskills

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// The component's declared settings (package.json `settings`): what a member
// and their organization add to the shipped policy. bnbloc stores and
// delivers them; the desktop writes them to the settings file it names at
// launch (an owner-only JSON object by key: a `lines` setting is an array of
// strings, any other a string; a setting without a value is absent).
const (
	// Member settings.
	SettingExtraSkillDirs = "extraSkillDirs"
	SettingExcludes       = "excludes"
	// Organization settings.
	SettingSecretNames    = "secretNames"
	SettingSecretPatterns = "secretPatterns"
	SettingMaxSkills      = "maxSkills"
)

// orgPatternName names a content rule an organization added: the reason a
// skill is not shared says it.
const orgPatternName = "a pattern your organization set"

// ComponentSettings are the declared settings' values.
type ComponentSettings struct {
	ExtraSkillDirs []string
	Excludes       []string
	SecretNames    []string
	SecretPatterns []string
	// MaxSkills is the number's text ("" = the policy's own limit).
	MaxSkills string
}

// ReadComponentSettings reads the settings file. An empty path is no
// settings (the component was started without one); a file that is named but
// cannot be read or parsed is an error, never ignored.
func ReadComponentSettings(path string) (ComponentSettings, error) {
	var cs ComponentSettings
	if path == "" {
		return cs, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return cs, fmt.Errorf("component settings file: %w", err)
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return cs, errors.New("component settings file: not a JSON object")
	}
	lines := func(key string, dst *[]string) error {
		v, ok := values[key]
		if !ok {
			return nil
		}
		var list []string
		if err := json.Unmarshal(v, &list); err != nil {
			return fmt.Errorf("component settings file: %s is not a list of lines", key)
		}
		for _, l := range list {
			if l = strings.TrimSpace(l); l != "" {
				*dst = append(*dst, l)
			}
		}
		return nil
	}
	for key, dst := range map[string]*[]string{
		SettingExtraSkillDirs: &cs.ExtraSkillDirs,
		SettingExcludes:       &cs.Excludes,
		SettingSecretNames:    &cs.SecretNames,
		SettingSecretPatterns: &cs.SecretPatterns,
	} {
		if err := lines(key, dst); err != nil {
			return ComponentSettings{}, err
		}
	}
	if v, ok := values[SettingMaxSkills]; ok {
		if err := json.Unmarshal(v, &cs.MaxSkills); err != nil {
			return ComponentSettings{}, fmt.Errorf("component settings file: %s is not text", SettingMaxSkills)
		}
		cs.MaxSkills = strings.TrimSpace(cs.MaxSkills)
	}
	return cs, nil
}

// Overlay is the settings as a policy to merge onto the shipped one, and a
// note for each value it left out (the member reads them in the state). A
// value that would weaken a check is an error, not a note: a pattern that
// does not compile, a skill count that is not a positive number.
func (cs ComponentSettings) Overlay(home string) (Policy, []string, error) {
	var o Policy
	var notes []string
	for _, dir := range cs.ExtraSkillDirs {
		abs := filepath.Clean(ExpandHome(dir, home))
		if !filepath.IsAbs(abs) || abs == filepath.Clean(home) || !within(home, abs) {
			notes = append(notes, fmt.Sprintf("the extra agent skill folder %s is left out: it is not a folder under your home", dir))
			continue
		}
		o.AgentDirs = append(o.AgentDirs, dir)
	}
	o.Exclude = cs.Excludes
	o.SecretNames = cs.SecretNames
	for _, pat := range cs.SecretPatterns {
		if _, err := regexp.Compile(pat); err != nil {
			return Policy{}, nil, fmt.Errorf("the organization's secret pattern %q does not compile: %w", pat, err)
		}
		o.SecretRules = append(o.SecretRules, SecretRule{Name: orgPatternName, Pattern: pat})
	}
	if cs.MaxSkills != "" {
		n, err := strconv.Atoi(cs.MaxSkills)
		if err != nil || n <= 0 {
			return Policy{}, nil, fmt.Errorf("the setting %s (skills in a sandbox) must be a positive whole number, got %q", SettingMaxSkills, cs.MaxSkills)
		}
		o.Limits.Skills = n
	}
	return o, notes, nil
}

// LoadPolicy is the policy the component works to: the shipped default with
// the settings file's values merged onto it (Policy.Merge, the one merge),
// and the notes on what the settings left out.
func LoadPolicy(settingsFile, home string) (Policy, []string, error) {
	cs, err := ReadComponentSettings(settingsFile)
	if err != nil {
		return Policy{}, nil, err
	}
	o, notes, err := cs.Overlay(home)
	if err != nil {
		return Policy{}, nil, err
	}
	return DefaultPolicy().Merge(o), notes, nil
}
