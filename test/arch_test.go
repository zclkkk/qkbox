package test

import (
	"strings"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestArchitectureBoundaries(t *testing.T) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps,
		Dir:  "../",
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		t.Fatalf("Failed to load packages: %v", err)
	}

	for _, pkg := range pkgs {
		for importPath := range pkg.Imports {
			if strings.HasPrefix(importPath, "github.com/sagernet/sing") {
				if !strings.Contains(pkg.PkgPath, "internal/singboxadapter") {
					t.Errorf("Architecture violation: Package %s imports %s. Only internal/singboxadapter is allowed to import sing-box/sing packages.", pkg.PkgPath, importPath)
				}
			}
		}
	}
}
