// Package app coordinates definition files, request execution, and local runtime state.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"apitool/internal/auth"
	"apitool/internal/collection"
	localgit "apitool/internal/git"
	"apitool/internal/model"
	"apitool/internal/resolve"
	"apitool/internal/runtime"
	"apitool/internal/transport"
	"apitool/internal/validate"
	"apitool/internal/workspace"
)

// Dependencies permits tests and embedding applications to replace the two external boundaries.
type Dependencies struct {
	TokenProvider auth.TokenProvider
	OpenRuntime   func(string) (*runtime.Store, error)
	OpenGit       func(string) GitRepository
	Execute       func(context.Context, model.EffectiveRequest, auth.TokenProvider) (model.Response, *model.ExecutionError)
}

// GitRepository is the safe, text-oriented Git boundary used by the app.
// Implementations must not stage files as a side effect of these methods.
type GitRepository interface {
	Status(context.Context) (string, error)
	Diff(context.Context) (string, error)
	Pull(context.Context) (string, error)
	Push(context.Context) (string, error)
	Commit(context.Context, string) (string, error)
}

type Service struct {
	deps             Dependencies
	opened           *Workspace
	store            *runtime.Store
	git              GitRepository
	confirmDangerous bool
}

type OpenOptions struct {
	Collection       string
	Environment      string
	ConfirmDangerous bool
}

type Workspace struct {
	Root             string
	Collections      map[string]CollectionView
	ActiveCollection string
}
type UIPreferences struct {
	ActiveCollection string
	ExplorerWidth    int
}
type CollectionView struct {
	Path         string
	Root         string
	Collection   model.Collection
	Environments map[string]model.Environment
	Environment  string
	Tree         collection.Tree
	Diagnostics  []model.Diagnostic
}
type Selection struct {
	Collection  string
	Environment string
	RequestID   string
	Confirmed   bool
}
type SendResult struct {
	Response             *model.Response
	ExecutionError       *model.ExecutionError
	Diagnostics          []model.Diagnostic
	ConfirmationRequired bool
	Logs                 []runtime.LogEntry
}
type DeleteTarget struct {
	Collection string
	RequestID  string
	Group      bool
	Paths      []string
}

func New(deps Dependencies) (*Service, error) {
	if deps.TokenProvider == nil {
		deps.TokenProvider = auth.NewClientCredentials(nil)
	}
	if deps.OpenRuntime == nil {
		deps.OpenRuntime = runtime.Open
	}
	if deps.OpenGit == nil {
		deps.OpenGit = func(root string) GitRepository { return localgit.New(root, exec.CommandContext) }
	}
	if deps.Execute == nil {
		deps.Execute = transport.Execute
	}
	return &Service{deps: deps}, nil
}

func (s *Service) OpenWorkspace(_ context.Context, start string, options OpenOptions) (Workspace, error) {
	root, err := workspace.FindRoot(start)
	if err != nil {
		return Workspace{}, err
	}
	store, err := s.deps.OpenRuntime(root)
	if err != nil {
		return Workspace{}, err
	}
	state, err := store.LoadState()
	if err != nil {
		return Workspace{}, err
	}
	activeCollection := options.Collection
	if activeCollection == "" {
		activeCollection = state.ActiveCollection
	}
	opened := Workspace{Root: root, Collections: map[string]CollectionView{}, ActiveCollection: activeCollection}
	for _, found := range workspace.Discover(root) {
		view := CollectionView{Path: found.Path, Root: found.Root, Collection: found.Collection, Diagnostics: append([]model.Diagnostic(nil), found.Diagnostics...), Environments: loadEnvironments(found.Root)}
		var treeDiagnostics []model.Diagnostic
		view.Tree, treeDiagnostics = collection.BuildTree(found.Root)
		view.Diagnostics = append(view.Diagnostics, treeDiagnostics...)
		environment := state.LastActiveEnvironment[found.Path]
		if options.Environment != "" && options.Collection == found.Path {
			environment = options.Environment
		}
		if environment != "" {
			if _, ok := view.Environments[environment]; !ok && options.Environment != "" && options.Collection == found.Path {
				return Workspace{}, fmt.Errorf("collection %q has no environment %q", found.Path, environment)
			}
			if _, ok := view.Environments[environment]; ok {
				view.Environment = environment
			}
		}
		opened.Collections[found.Path] = view
	}
	if opened.ActiveCollection != "" {
		if _, ok := opened.Collections[opened.ActiveCollection]; !ok {
			if options.Collection != "" {
				return Workspace{}, fmt.Errorf("collection %q not found", options.Collection)
			}
			opened.ActiveCollection = ""
		}
	}
	s.opened, s.store, s.git, s.confirmDangerous = &opened, store, s.deps.OpenGit(root), options.ConfirmDangerous
	return opened, nil
}

func (s *Service) GitStatus(ctx context.Context) (string, error) {
	repo, err := s.repository()
	if err != nil {
		return "", err
	}
	return repo.Status(ctx)
}

func (s *Service) GitDiff(ctx context.Context) (string, error) {
	repo, err := s.repository()
	if err != nil {
		return "", err
	}
	return repo.Diff(ctx)
}

func (s *Service) GitPull(ctx context.Context) (string, error) {
	repo, err := s.repository()
	if err != nil {
		return "", err
	}
	return repo.Pull(ctx)
}

func (s *Service) GitPush(ctx context.Context) (string, error) {
	repo, err := s.repository()
	if err != nil {
		return "", err
	}
	return repo.Push(ctx)
}

func (s *Service) GitCommit(ctx context.Context, message string) (string, error) {
	repo, err := s.repository()
	if err != nil {
		return "", err
	}
	return repo.Commit(ctx, message)
}

// TreeDiagnostics returns request-tree errors alongside collection load errors.
func (v CollectionView) TreeDiagnostics() []model.Diagnostic {
	var result []model.Diagnostic
	for _, invalid := range v.Tree.Invalid {
		result = append(result, invalid.Diagnostics...)
	}
	return result
}

func (s *Service) OpenCollection(_ context.Context, collectionPath string) (CollectionView, error) {
	return s.collection(collectionPath)
}

// Workspace returns the currently opened workspace view for read-only clients
// such as the terminal UI.
func (s *Service) Workspace() (Workspace, error) {
	if s.opened == nil {
		return Workspace{}, errors.New("workspace is not open")
	}
	return *s.opened, nil
}

func (s *Service) UIPreferences() (UIPreferences, error) {
	if s.store == nil {
		return UIPreferences{}, errors.New("workspace is not open")
	}
	state, err := s.store.LoadState()
	if err != nil {
		return UIPreferences{}, err
	}
	return UIPreferences{ActiveCollection: state.ActiveCollection, ExplorerWidth: state.ExplorerWidth}, nil
}
func (s *Service) SaveUIPreferences(collection string, explorerWidth int) error {
	if s.store == nil {
		return errors.New("workspace is not open")
	}
	return s.store.SaveState(runtime.State{ActiveCollection: collection, ExplorerWidth: explorerWidth})
}

func (s *Service) SelectEnvironment(_ context.Context, collectionPath, environment string) (CollectionView, error) {
	view, err := s.collection(collectionPath)
	if err != nil {
		return CollectionView{}, err
	}
	if _, ok := view.Environments[environment]; !ok {
		return CollectionView{}, fmt.Errorf("collection %q has no environment %q", collectionPath, environment)
	}
	if err := s.store.SaveState(runtime.State{LastActiveEnvironment: map[string]string{collectionPath: environment}}); err != nil {
		return CollectionView{}, err
	}
	view.Environment = environment
	s.opened.Collections[collectionPath] = view
	return view, nil
}

func (s *Service) SaveRequest(_ context.Context, selection Selection, request model.Request) error {
	if diagnostics := validate.Definition(request); len(diagnostics) != 0 {
		return ValidationError{Diagnostics: diagnostics}
	}
	view, err := s.collection(selection.Collection)
	if err != nil {
		return err
	}
	path, err := requestPath(view.Root, selection.RequestID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create request directory: %w", err)
	}
	if err := collection.SaveRequest(path, request); err != nil {
		return err
	}
	return s.refreshTree(selection.Collection)
}

func (s *Service) Send(ctx context.Context, selection Selection) SendResult {
	view, err := s.collection(selection.Collection)
	if err != nil {
		return executionResult(err)
	}
	if selection.Environment == "" {
		selection.Environment = view.Environment
	}
	env, ok := view.Environments[selection.Environment]
	if !ok {
		return executionResult(fmt.Errorf("collection %q has no environment %q", selection.Collection, selection.Environment))
	}
	node, ok := view.Tree.Requests[selection.RequestID]
	if !ok {
		return executionResult(fmt.Errorf("request %q not found", selection.RequestID))
	}
	if diagnostics := validate.Definition(node.Request); len(diagnostics) != 0 {
		return SendResult{Diagnostics: diagnostics, ExecutionError: &model.ExecutionError{Stage: model.StageRequestBuild, Category: model.CategoryRequestBuild, SafeMessage: "Request definition is invalid"}}
	}
	if s.confirmDangerous && dangerous(node.Request.Method) && !selection.Confirmed {
		return SendResult{ConfirmationRequired: true, ExecutionError: &model.ExecutionError{Stage: model.StageRequestBuild, Category: model.CategoryRequestBuild, SafeMessage: "Request confirmation required"}}
	}
	groups := make([]model.Group, len(node.Groups))
	for i := range node.Groups {
		groups[i] = node.Groups[i].Group
	}
	effective, diagnostics := resolve.Effective(view.Collection, env, groups, node.Request)
	if hasErrors(diagnostics) {
		return SendResult{Diagnostics: diagnostics, ExecutionError: &model.ExecutionError{Stage: model.StageRequestBuild, Category: model.CategoryRequestBuild, SafeMessage: "Request definition is invalid"}}
	}
	response, execErr := s.deps.Execute(ctx, effective, s.deps.TokenProvider)
	key := runtime.Key{CollectionPath: selection.Collection, Environment: selection.Environment, RequestID: selection.RequestID}
	log := executionLog(key, effective, response, execErr)
	_ = s.store.AppendLog(log)
	_ = s.store.AppendHistory(key, effective.Method, response, execErr)
	if execErr == nil {
		_ = s.store.SaveResponse(key, response)
		return SendResult{Response: &response, Logs: []runtime.LogEntry{log}}
	}
	return SendResult{ExecutionError: execErr, Logs: []runtime.LogEntry{log}}
}

func (s *Service) DuplicateRequest(ctx context.Context, source Selection, destinationID string) (Selection, error) {
	view, err := s.collection(source.Collection)
	if err != nil {
		return Selection{}, err
	}
	node, ok := view.Tree.Requests[source.RequestID]
	if !ok {
		return Selection{}, fmt.Errorf("request %q not found", source.RequestID)
	}
	path, err := requestPath(view.Root, destinationID)
	if err != nil {
		return Selection{}, err
	}
	if _, err := os.Lstat(path); err == nil {
		return Selection{}, fmt.Errorf("destination request %q already exists", destinationID)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Selection{}, err
	}
	if err := s.SaveRequest(ctx, Selection{Collection: source.Collection, RequestID: destinationID}, node.Request); err != nil {
		return Selection{}, err
	}
	return Selection{Collection: source.Collection, Environment: source.Environment, RequestID: destinationID}, nil
}

// Delete returns exact affected paths without mutation until confirmed is true.
func (s *Service) Delete(_ context.Context, collectionPath, id string, group, confirmed bool) (DeleteTarget, error) {
	view, err := s.collection(collectionPath)
	if err != nil {
		return DeleteTarget{}, err
	}
	base := filepath.Join(view.Root, ".api", "requests")
	rel := filepath.FromSlash(id)
	if id == "" || filepath.IsAbs(rel) || strings.HasPrefix(filepath.Clean(rel), "..") {
		return DeleteTarget{}, errors.New("invalid deletion target")
	}
	safeRelative := rel
	if !group {
		safeRelative += ".yaml"
	}
	if err := rejectSymlinkSegments(base, safeRelative); err != nil {
		return DeleteTarget{}, err
	}
	target := filepath.Join(base, rel)
	if !group {
		target += ".yaml"
	}
	paths, err := deletionPaths(target, group)
	if err != nil {
		return DeleteTarget{}, err
	}
	result := DeleteTarget{Collection: collectionPath, RequestID: id, Group: group, Paths: paths}
	if confirmed {
		if group {
			err = os.RemoveAll(target)
		} else {
			err = os.Remove(target)
		}
		if err != nil {
			return DeleteTarget{}, err
		}
	}
	return result, nil
}

type ValidationError struct{ Diagnostics []model.Diagnostic }

func (e ValidationError) Error() string { return "request definition is invalid" }

func (s *Service) collection(path string) (CollectionView, error) {
	if s.opened == nil {
		return CollectionView{}, errors.New("workspace is not open")
	}
	view, ok := s.opened.Collections[path]
	if !ok {
		return CollectionView{}, fmt.Errorf("collection %q not found", path)
	}
	return view, nil
}

func (s *Service) repository() (GitRepository, error) {
	if s.opened == nil || s.git == nil {
		return nil, errors.New("workspace is not open")
	}
	return s.git, nil
}
func (s *Service) refreshTree(collectionPath string) error {
	view, err := s.collection(collectionPath)
	if err != nil {
		return err
	}
	tree, diagnostics := collection.BuildTree(view.Root)
	view.Tree = tree
	view.Diagnostics = diagnostics
	s.opened.Collections[collectionPath] = view
	return nil
}
func loadEnvironments(root string) map[string]model.Environment {
	result := map[string]model.Environment{}
	entries, err := os.ReadDir(filepath.Join(root, ".api", "environments"))
	if err != nil {
		return result
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		if env, err := collection.LoadEnvironment(filepath.Join(root, ".api", "environments", entry.Name())); err == nil {
			result[name] = env
		}
	}
	return result
}
func requestPath(root, id string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(id))
	if id == "" || filepath.IsAbs(clean) || clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("invalid request ID")
	}
	return filepath.Join(root, ".api", "requests", clean+".yaml"), nil
}
func dangerous(method string) bool {
	switch strings.ToUpper(method) {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}
func hasErrors(diags []model.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == model.SeverityError {
			return true
		}
	}
	return false
}
func executionResult(err error) SendResult {
	return SendResult{ExecutionError: &model.ExecutionError{Stage: model.StageRequestBuild, Category: model.CategoryRequestBuild, SafeMessage: err.Error()}}
}
func executionLog(key runtime.Key, e model.EffectiveRequest, r model.Response, x *model.ExecutionError) runtime.LogEntry {
	path := ""
	if parsed, err := url.Parse(e.URL); err == nil {
		query := parsed.Query()
		for name := range query {
			if sensitiveLogKey(name) {
				query.Set(name, "[REDACTED]")
			}
		}
		parsed.RawQuery = query.Encode()
		path = parsed.RequestURI()
	}
	entry := runtime.LogEntry{Key: key, Method: e.Method, Path: path, StatusCode: r.StatusCode, Duration: r.Duration}
	if x != nil {
		entry.ErrorCategory = x.Category
	}
	return entry
}
func sensitiveLogKey(name string) bool {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(name))
	return strings.Contains(normalized, "token") || strings.Contains(normalized, "secret") || strings.Contains(normalized, "apikey") || normalized == "authorization" || normalized == "cookie"
}
func rejectSymlinkSegments(base, relative string) error {
	current := base
	for _, segment := range strings.Split(filepath.Clean(relative), string(filepath.Separator)) {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("deletion target must not traverse a symlink")
		}
	}
	return nil
}
func deletionPaths(target string, group bool) ([]string, error) {
	info, err := os.Lstat(target)
	if err != nil {
		return nil, err
	}
	if !group || !info.IsDir() {
		return []string{target}, nil
	}
	var paths []string
	err = filepath.WalkDir(target, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		paths = append(paths, path)
		return nil
	})
	sort.Strings(paths)
	return paths, err
}
