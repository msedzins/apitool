// Package workspace locates Git workspaces and their API collections.
package workspace

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"apitool/internal/collection"
	"apitool/internal/model"
)

// FindRoot returns the nearest ancestor that contains a normal Git directory
// or a linked-worktree .git file.
func FindRoot(start string) (string, error) {
	path, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(path); err != nil {
		return "", err
	} else if !info.IsDir() {
		path = filepath.Dir(path)
	}

	for {
		gitPath := filepath.Join(path, ".git")
		info, err := os.Lstat(gitPath)
		if err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return path, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", fmt.Errorf("git workspace not found from %q", start)
		}
		path = parent
	}
}

// Discover finds every directory below root that directly contains .api.
// Malformed collection metadata remains discoverable with a diagnostic.
func Discover(root string) []collection.LocatedCollection {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil
	}
	var collections []collection.LocatedCollection
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.Type()&fs.ModeSymlink != 0 {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root {
			switch entry.Name() {
			case ".git", ".apitool", ".api":
				return filepath.SkipDir
			}
		}
		if path == root {
			return nil
		}
		apiPath := filepath.Join(path, ".api")
		apiInfo, err := os.Lstat(apiPath)
		if err != nil || !apiInfo.IsDir() || apiInfo.Mode()&fs.ModeSymlink != 0 {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		located := collection.LocatedCollection{Root: path, Path: filepath.ToSlash(relative)}
		metadataPath := filepath.Join(apiPath, "collection.yaml")
		metadata, err := collection.LoadCollectionMeta(metadataPath)
		if err != nil {
			located.Diagnostics = []model.Diagnostic{collectionLoadDiagnostic(filepath.ToSlash(filepath.Join(relative, ".api", "collection.yaml")), err)}
		} else {
			located.Collection = metadata
		}
		collections = append(collections, located)
		return nil
	})
	sort.Slice(collections, func(i, j int) bool { return collections[i].Path < collections[j].Path })
	return collections
}

func collectionLoadDiagnostic(path string, err error) model.Diagnostic {
	return model.Diagnostic{
		Code:     "collection_load",
		Path:     path,
		Message:  fmt.Sprintf("load collection metadata: %v", err),
		Severity: model.SeverityError,
	}
}
