package localskills

import (
	"archive/tar"
	"bytes"
	"fmt"
	"os"
	"sort"

	"github.com/luckylhb90/bnbloc/component"
	"github.com/luckylhb90/bnbloc/core"
)

// Pack is one skill as a tar the Library imports (<name>/SKILL.md and the
// rest): only the files the policy lets through, none of it if the skill is not
// eligible (a credential, a link out, too big) or holds a file name the library
// refuses. The archive is the library's bundle format: uncompressed ustar,
// sorted, every path a core.ValidBundlePath, within the content caps.
func (s *Scanner) Pack(sk Skill) ([]byte, error) {
	if !sk.Eligible {
		return nil, fmt.Errorf("the skill %s cannot be packed: %s", sk.Name, sk.Reason)
	}
	if sk.PublishProblem != "" {
		return nil, fmt.Errorf("the skill %s cannot be added to the library: %s", sk.Name, sk.PublishProblem)
	}
	files, err := s.Files(sk.Dir)
	if err != nil {
		return nil, err
	}
	bundle := make(map[string][]byte, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f.Abs)
		if err != nil {
			return nil, err
		}
		bundle[sk.Name+"/"+f.Rel] = data
	}
	return packBundle(bundle)
}

// packBundle writes files as a deterministic ustar archive (sorted paths,
// mode 0644, no times), refusing what the library would.
func packBundle(files map[string][]byte) ([]byte, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("bundle holds no files")
	}
	if len(files) > component.ContentMaxFiles {
		return nil, fmt.Errorf("bundle holds more than %d files", component.ContentMaxFiles)
	}
	paths := make([]string, 0, len(files))
	total := 0
	for p, data := range files {
		if !core.ValidBundlePath(p) {
			return nil, fmt.Errorf("%s is not a valid bundle path ([A-Za-z0-9._-] segments)", p)
		}
		if total += len(data); total > component.ContentMaxBytes {
			return nil, fmt.Errorf("bundle exceeds %d bytes", component.ContentMaxBytes)
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, rel := range paths {
		data := files[rel]
		if err := tw.WriteHeader(&tar.Header{Name: rel, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}); err != nil {
			return nil, err
		}
		if _, err := tw.Write(data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
