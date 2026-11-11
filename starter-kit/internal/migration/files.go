package migration

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const Directory = "migrations/sql"

var filename = regexp.MustCompile(`^([0-9]+)_([a-zA-Z0-9_-]+)\.sql$`)

func Version(name string) (int64, error) {
	matches := filename.FindStringSubmatch(name)
	if matches == nil {
		return 0, fmt.Errorf("invalid migration filename %q; expected version_name.sql", name)
	}
	version, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("invalid migration version in %q", name)
	}
	return version, nil
}
func NextPath(name string) (string, error) {
	paths, err := filepath.Glob(filepath.Join(Directory, "*.sql"))
	if err != nil {
		return "", err
	}
	version, err := strconv.ParseInt(time.Now().UTC().Format("20060102150405"), 10, 64)
	if err != nil {
		return "", err
	}
	for _, path := range paths {
		current, err := Version(filepath.Base(path))
		if err != nil {
			return "", err
		}
		if current >= version {
			if current == int64(^uint64(0)>>1) {
				return "", fmt.Errorf("migration version overflow")
			}
			version = current + 1
		}
	}
	return filepath.Join(Directory, fmt.Sprintf("%d_%s.sql", version, name)), nil
}
func Sections(content string) (string, string, error) {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<<<<<<<") || strings.HasPrefix(trimmed, ">>>>>>>") || trimmed == "=======" {
			return "", "", fmt.Errorf("unresolved Git conflict markers")
		}
	}
	up, down := "-- +goose Up", "-- +goose Down"
	if strings.Count(content, up) != 1 || strings.Count(content, down) != 1 {
		return "", "", fmt.Errorf("exactly one goose Up and Down section is required")
	}
	start, end := strings.Index(content, up), strings.Index(content, down)
	if start > end {
		return "", "", fmt.Errorf("Up must precede Down")
	}
	u, d := strings.TrimSpace(content[start+len(up):end]), strings.TrimSpace(content[end+len(down):])
	if u == "" || d == "" {
		return "", "", fmt.Errorf("Up and Down must contain SQL")
	}
	return u, d, nil
}
func Check() error {
	paths, err := filepath.Glob(filepath.Join(Directory, "*.sql"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no migrations in %s", Directory)
	}
	seen := map[int64]string{}
	problems := []string{}
	for _, path := range paths {
		version, err := Version(filepath.Base(path))
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		if previous, ok := seen[version]; ok {
			problems = append(problems, fmt.Sprintf("duplicate version %d: %s and %s", version, previous, path))
		}
		seen[version] = path
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if _, _, err := Sections(string(raw)); err != nil {
			problems = append(problems, fmt.Sprintf("%s: %v", path, err))
		}
		if strings.Contains(string(raw), "MODKIT_REVIEW_REQUIRED") {
			problems = append(problems, path+": unfinished migration")
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "\n"))
	}
	return nil
}
func Draft() []byte {
	return []byte(`-- +goose Up
-- MODKIT_REVIEW_REQUIRED: replace this guard with reviewed forward SQL.
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Implement and review this migration before applying it'; END $$;
-- +goose StatementEnd

-- +goose Down
-- MODKIT_REVIEW_REQUIRED: implement a reviewed rollback (or explicitly reject rollback).
-- +goose StatementBegin
DO $$ BEGIN RAISE EXCEPTION 'Rollback has not been implemented'; END $$;
-- +goose StatementEnd
`)
}

// Merge creates a review artifact OUTSIDE the executable migration directory.
// Source order controls Up ordering; Down sections are reversed.
// It never rewrites history, removes originals or guesses SQL conflict resolution.
func Merge(sources []string) ([]byte, error) {
	if len(sources) < 2 {
		return nil, fmt.Errorf("at least two --source filenames are required")
	}
	ups, downs, refs := []string{}, []string{}, []string{}
	seen := map[string]bool{}
	for _, source := range sources {
		if filepath.Base(source) != source {
			return nil, fmt.Errorf("--source must be a filename inside %s", Directory)
		}
		if seen[source] {
			return nil, fmt.Errorf("duplicate source %q", source)
		}
		seen[source] = true
		if _, err := Version(source); err != nil {
			return nil, err
		}
		raw, err := os.ReadFile(filepath.Join(Directory, source))
		if err != nil {
			return nil, err
		}
		up, down, err := Sections(string(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", source, err)
		}
		refs = append(refs, fmt.Sprintf("-- source %s sha256 %x", source, sha256.Sum256(raw)))
		ups = append(ups, "-- From "+source+"\n"+up)
		downs = append([]string{"-- From " + source + "\n" + down}, downs...)
	}
	return []byte("-- MODKIT_REVIEW_REQUIRED: draft only; resolve SQL conflicts and inspect applied history in every environment.\n" +
		strings.Join(refs, "\n") + "\n-- +goose Up\n" + strings.Join(ups, "\n\n") + "\n\n-- +goose Down\n" + strings.Join(downs, "\n\n") + "\n"), nil
}
func Names() ([]string, error) {
	paths, err := filepath.Glob(filepath.Join(Directory, "*.sql"))
	sort.Strings(paths)
	return paths, err
}
