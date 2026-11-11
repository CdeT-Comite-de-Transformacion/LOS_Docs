package scaffold

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// Write validates the complete file plan before changing any files.
// Dry runs execute the same validation without creating directories or files.
func Write(files map[string][]byte, overwrite map[string]bool, dry bool, out io.Writer) error {
	paths := make([]string, 0, len(files))
	actions := map[string]string{}
	for path, content := range files {
		paths = append(paths, path)
		old, err := os.ReadFile(path)
		switch {
		case err == nil:
			if !overwrite[path] {
				return fmt.Errorf("%s already exists", path)
			}
			actions[path] = "UPDATE"
			if bytes.Equal(old, content) {
				actions[path] = "UNCHANGED"
			}
		case os.IsNotExist(err):
			actions[path] = "CREATE"
		default:
			return err
		}
	}
	sort.Strings(paths)
	if dry {
		for _, path := range paths {
			if out != nil {
				fmt.Fprintf(out, "%s %s\n", actions[path], path)
			}
		}
		return nil
	}
	for _, path := range paths {
		if actions[path] == "UNCHANGED" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, files[path], 0644); err != nil {
			return err
		}
	}
	return nil
}
