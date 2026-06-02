package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSharedPackagesDoNotReferenceSingBox(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range []string{"shared/api", "shared/model"} {
		dir := filepath.Join(root, rel)
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			if strings.Contains(strings.ToLower(string(content)), "sing-box") || strings.Contains(string(content), "github.com/sagernet/sing-box") {
				t.Fatalf("%s must not reference sing-box", path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
