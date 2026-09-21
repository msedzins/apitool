package collection

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"apitool/internal/model"
	"apitool/internal/validate"
)

// LocatedCollection identifies a collection found below a workspace root.
// Path is slash-separated and relative to that workspace.
type LocatedCollection struct {
	Root        string
	Path        string
	Collection  model.Collection
	Diagnostics []model.Diagnostic
}

// Tree is the loaded, deterministic request view for a collection.
type Tree struct {
	Groups     []GroupNode
	Requests   map[string]RequestNode
	RequestIDs []string
	Invalid    map[string]InvalidNode
}

// GroupNode represents one physical directory below .api/requests.
type GroupNode struct {
	ID          string
	ParentID    string
	Group       model.Group
	Diagnostics []model.Diagnostic
}

// RequestNode contains a request and its outermost-to-innermost group chain.
type RequestNode struct {
	ID          string
	Request     model.Request
	Groups      []GroupNode
	Diagnostics []model.Diagnostic
}

// InvalidNode remains addressable in the explorer when a definition cannot load.
type InvalidNode struct {
	ID          string
	Path        string
	Diagnostics []model.Diagnostic
}

type loadedRequest struct {
	id      string
	path    string
	request model.Request
}

// BuildTree loads all request definitions below collectionRoot. Bad definitions
// become invalid nodes and diagnostics without preventing their siblings loading.
func BuildTree(collectionRoot string) (Tree, []model.Diagnostic) {
	tree := Tree{
		Requests: make(map[string]RequestNode),
		Invalid:  make(map[string]InvalidNode),
	}
	requestsRoot := filepath.Join(collectionRoot, ".api", "requests")
	groups := make(map[string]GroupNode)
	var requests []loadedRequest
	var diagnostics []model.Diagnostic
	if info, err := os.Lstat(requestsRoot); err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		if err == nil {
			err = fmt.Errorf("requests directory must not be a symlink")
		}
		diagnostic := loadDiagnostic("tree_walk", relativeDefinitionPath(collectionRoot, requestsRoot), err)
		addInvalid(&tree, "", requestsRoot, diagnostic)
		return tree, []model.Diagnostic{diagnostic}
	}

	err := filepath.WalkDir(requestsRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			id := relativeID(requestsRoot, path)
			diagnostic := loadDiagnostic("tree_walk", relativeDefinitionPath(collectionRoot, path), walkErr)
			diagnostics = append(diagnostics, diagnostic)
			addInvalid(&tree, id, path, diagnostic)
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path == requestsRoot {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".apitool", ".api":
				return filepath.SkipDir
			}
			id := relativeID(requestsRoot, path)
			groups[id] = GroupNode{ID: id, ParentID: parentGroupID(id)}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".yaml" {
			return nil
		}

		id := requestID(requestsRoot, path)
		if entry.Name() == "_group.yaml" {
			groupID := relativeID(requestsRoot, filepath.Dir(path))
			node := groups[groupID]
			group, err := LoadGroup(path)
			if err != nil {
				diagnostic := loadDiagnostic("group_load", relativeDefinitionPath(collectionRoot, path), err)
				node.Diagnostics = append(node.Diagnostics, diagnostic)
				diagnostics = append(diagnostics, diagnostic)
				addInvalid(&tree, groupID, path, diagnostic)
			} else {
				node.Group = group
				groupDiagnostics := diagnosticsAt(relativeDefinitionPath(collectionRoot, path), validate.Auth(group.Auth))
				if len(groupDiagnostics) != 0 {
					node.Diagnostics = append(node.Diagnostics, groupDiagnostics...)
					diagnostics = append(diagnostics, groupDiagnostics...)
					for _, diagnostic := range groupDiagnostics {
						addInvalid(&tree, groupID, path, diagnostic)
					}
				}
			}
			groups[groupID] = node
			return nil
		}

		request, err := LoadRequest(path)
		if err != nil {
			diagnostic := loadDiagnostic("request_load", relativeDefinitionPath(collectionRoot, path), err)
			diagnostics = append(diagnostics, diagnostic)
			addInvalid(&tree, id, path, diagnostic)
			return nil
		}
		requestDiagnostics := diagnosticsAt(relativeDefinitionPath(collectionRoot, path), validate.Definition(request))
		if len(requestDiagnostics) != 0 {
			diagnostics = append(diagnostics, requestDiagnostics...)
			for _, diagnostic := range requestDiagnostics {
				addInvalid(&tree, id, path, diagnostic)
			}
			return nil
		}
		requests = append(requests, loadedRequest{id: id, path: path, request: request})
		return nil
	})
	if err != nil {
		diagnostic := loadDiagnostic("tree_walk", relativeDefinitionPath(collectionRoot, requestsRoot), err)
		diagnostics = append(diagnostics, diagnostic)
		addInvalid(&tree, "", requestsRoot, diagnostic)
	}

	for _, node := range groups {
		tree.Groups = append(tree.Groups, node)
	}
	for _, loaded := range requests {
		chain := groupChain(groups, filepath.Dir(loaded.path), requestsRoot)
		node := RequestNode{ID: loaded.id, Request: loaded.request, Groups: chain}
		for _, group := range chain {
			for _, diagnostic := range group.Diagnostics {
				node.Diagnostics = append(node.Diagnostics, diagnostic)
				addInvalid(&tree, loaded.id, loaded.path, diagnostic)
			}
		}
		tree.Requests[loaded.id] = node
		tree.RequestIDs = append(tree.RequestIDs, loaded.id)
	}
	sort.Slice(tree.Groups, func(i, j int) bool { return tree.Groups[i].ID < tree.Groups[j].ID })
	sort.Strings(tree.RequestIDs)
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path == diagnostics[j].Path {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Path < diagnostics[j].Path
	})
	return tree, diagnostics
}

func diagnosticsAt(definitionPath string, diagnostics []model.Diagnostic) []model.Diagnostic {
	for index := range diagnostics {
		if diagnostics[index].Path == "" {
			diagnostics[index].Path = definitionPath
		} else {
			diagnostics[index].Path = definitionPath + ":" + diagnostics[index].Path
		}
	}
	return diagnostics
}

func groupChain(groups map[string]GroupNode, requestDirectory, requestsRoot string) []GroupNode {
	relative, err := filepath.Rel(requestsRoot, requestDirectory)
	if err != nil || relative == "." {
		return nil
	}
	parts := strings.Split(filepath.ToSlash(relative), "/")
	chain := make([]GroupNode, 0, len(parts))
	for index := range parts {
		id := strings.Join(parts[:index+1], "/")
		if group, ok := groups[id]; ok {
			chain = append(chain, group)
		}
	}
	return chain
}

func relativeID(requestsRoot, path string) string {
	relative, err := filepath.Rel(requestsRoot, path)
	if err != nil || relative == "." {
		return ""
	}
	return filepath.ToSlash(relative)
}

func requestID(requestsRoot, path string) string {
	return strings.TrimSuffix(relativeID(requestsRoot, path), ".yaml")
}

func parentGroupID(id string) string {
	if index := strings.LastIndex(id, "/"); index >= 0 {
		return id[:index]
	}
	return ""
}

func relativeDefinitionPath(collectionRoot, path string) string {
	relative, err := filepath.Rel(collectionRoot, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(relative)
}

func loadDiagnostic(code, path string, err error) model.Diagnostic {
	return model.Diagnostic{
		Code:     code,
		Path:     path,
		Message:  fmt.Sprintf("load %s: %v", path, err),
		Severity: model.SeverityError,
	}
}

func addInvalid(tree *Tree, id, path string, diagnostic model.Diagnostic) {
	node := tree.Invalid[id]
	node.ID = id
	node.Path = path
	node.Diagnostics = append(node.Diagnostics, diagnostic)
	tree.Invalid[id] = node
}
