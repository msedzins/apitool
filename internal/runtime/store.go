// Package runtime stores local, non-definition data for an apitool workspace.
package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"apitool/internal/model"
)

var ErrNotFound = errors.New("runtime record not found")

// Key identifies a request execution within one collection environment.
type Key struct {
	CollectionPath string
	Environment    string
	RequestID      string
}

// Store owns the runtime directory for a single workspace.
type Store struct {
	root string
	mu   sync.Mutex
}

// State contains only non-secret local session preferences.
type State struct {
	LastActiveEnvironment map[string]string `json:"last_active_environment"`
	PanelPreferences      map[string]bool   `json:"panel_preferences"`
}

type cachedResponse struct {
	CollectionPath string        `json:"collection_path"`
	Environment    string        `json:"environment"`
	RequestID      string        `json:"request_id"`
	StatusCode     int           `json:"status_code"`
	Headers        http.Header   `json:"headers"`
	Body           []byte        `json:"body"`
	Duration       time.Duration `json:"duration_ns"`
	ReceivedAt     time.Time     `json:"received_at"`
}

// Open creates the visible, workspace-local runtime directory.
func Open(workspaceRoot string) (*Store, error) {
	if strings.TrimSpace(workspaceRoot) == "" {
		return nil, errors.New("workspace root is required")
	}
	root, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}
	store := &Store{root: filepath.Join(root, ".apitool")}
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		return nil, fmt.Errorf("create runtime directory: %w", err)
	}
	if err := os.Chmod(store.root, 0o700); err != nil {
		return nil, fmt.Errorf("secure runtime directory: %w", err)
	}
	logs := filepath.Join(store.root, "logs")
	if err := os.MkdirAll(logs, 0o700); err != nil {
		return nil, fmt.Errorf("create runtime logs directory: %w", err)
	}
	if err := os.Chmod(logs, 0o700); err != nil {
		return nil, fmt.Errorf("secure runtime logs directory: %w", err)
	}
	return store, nil
}

// SaveResponse stores the most recent completed response for key.
func (s *Store) SaveResponse(key Key, response model.Response) error {
	path, err := s.responsePath(key)
	if err != nil {
		return err
	}
	data, err := json.Marshal(cachedResponse{
		CollectionPath: key.CollectionPath,
		Environment:    key.Environment,
		RequestID:      key.RequestID,
		StatusCode:     response.StatusCode,
		Headers:        RedactHeaders(response.Headers),
		Body:           response.Body,
		Duration:       response.Duration,
		ReceivedAt:     response.ReceivedAt,
	})
	if err != nil {
		return fmt.Errorf("encode response cache: %w", err)
	}
	return atomicWrite(path, data)
}

// LatestResponse returns the last response saved for key.
func (s *Store) LatestResponse(key Key) (model.Response, error) {
	path, err := s.responsePath(key)
	if err != nil {
		return model.Response{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return model.Response{}, ErrNotFound
	}
	if err != nil {
		return model.Response{}, fmt.Errorf("read response cache: %w", err)
	}
	var cached cachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		return model.Response{}, fmt.Errorf("decode response cache: %w", err)
	}
	if cached.CollectionPath != key.CollectionPath || cached.Environment != key.Environment || cached.RequestID != key.RequestID {
		return model.Response{}, errors.New("response cache identity does not match key")
	}
	return model.Response{
		StatusCode: cached.StatusCode,
		Headers:    cached.Headers,
		Body:       cached.Body,
		Duration:   cached.Duration,
		ReceivedAt: cached.ReceivedAt,
	}, nil
}

// LoadState returns empty state on a workspace's first use.
func (s *Store) LoadState() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(filepath.Join(s.root, "state.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return emptyState(), nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read runtime state: %w", err)
	}
	state := emptyState()
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode runtime state: %w", err)
	}
	state.ensureMaps()
	if err := validateState(state); err != nil {
		return State{}, err
	}
	return state, nil
}

// SaveState atomically replaces the non-secret local session preferences.
func (s *Store) SaveState(state State) error {
	state.ensureMaps()
	if err := validateState(state); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode runtime state: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return atomicWrite(filepath.Join(s.root, "state.json"), data)
}

func emptyState() State {
	return State{LastActiveEnvironment: map[string]string{}, PanelPreferences: map[string]bool{}}
}

func (s *State) ensureMaps() {
	if s.LastActiveEnvironment == nil {
		s.LastActiveEnvironment = map[string]string{}
	}
	if s.PanelPreferences == nil {
		s.PanelPreferences = map[string]bool{}
	}
}

func validateState(state State) error {
	for collection, environment := range state.LastActiveEnvironment {
		if _, err := safePath(collection, "state collection path", true); err != nil {
			return err
		}
		if _, err := safePath(environment, "state environment", false); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) responsePath(key Key) (string, error) {
	segments, err := key.segments()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{s.root, "responses"}, append(segments, "latest.json")...)...), nil
}

func (k Key) segments() ([]string, error) {
	collection, err := safePath(k.CollectionPath, "collection path", true)
	if err != nil {
		return nil, err
	}
	environment, err := safePath(k.Environment, "environment", false)
	if err != nil {
		return nil, err
	}
	requestID, err := safePath(k.RequestID, "request ID", true)
	if err != nil {
		return nil, err
	}
	return append(append(collection, environment...), requestID...), nil
}

func safePath(value, label string, allowNested bool) ([]string, error) {
	if value == "" || filepath.IsAbs(value) || filepath.VolumeName(value) != "" || strings.Contains(value, `\`) {
		return nil, fmt.Errorf("invalid %s", label)
	}
	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || (!allowNested && strings.Contains(clean, string(filepath.Separator))) {
		return nil, fmt.Errorf("invalid %s", label)
	}
	parts := strings.Split(clean, string(filepath.Separator))
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return nil, fmt.Errorf("invalid %s", label)
		}
	}
	return parts, nil
}

func atomicWrite(path string, data []byte) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("create runtime path: %w", err)
	}
	if err := os.Chmod(parent, 0o700); err != nil {
		return fmt.Errorf("secure runtime path: %w", err)
	}
	file, err := os.CreateTemp(parent, ".write-*")
	if err != nil {
		return fmt.Errorf("create runtime temp file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("secure runtime temp file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("write runtime file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync runtime file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close runtime file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace runtime file: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("secure runtime file: %w", err)
	}
	return nil
}

func (s *Store) appendFile(name string, data []byte) error {
	path := filepath.Join(s.root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create runtime log path: %w", err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("secure runtime log path: %w", err)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open runtime file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return fmt.Errorf("secure runtime file: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return fmt.Errorf("append runtime file: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync runtime file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close runtime file: %w", err)
	}
	return nil
}
