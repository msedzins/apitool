// Package runtime stores local, non-definition data for an apitool workspace.
package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"apitool/internal/model"
	"golang.org/x/sys/unix"
)

var (
	ErrNotFound          = errors.New("runtime record not found")
	ErrSensitiveResponse = errors.New("response contains sensitive data and cannot be cached")
)

var temporarySequence atomic.Uint64

type Key struct{ CollectionPath, Environment, RequestID string }

// Store owns an opened .apitool descriptor. All writes below it use no-follow,
// descriptor-relative operations to keep runtime data in the workspace.
type Store struct {
	rootFD int
	mu     sync.Mutex
}

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

func Open(workspaceRoot string) (*Store, error) {
	if strings.TrimSpace(workspaceRoot) == "" {
		return nil, errors.New("workspace root is required")
	}
	root, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root: %w", err)
	}
	workspaceFD, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open workspace root: %w", err)
	}
	defer unix.Close(workspaceFD)
	runtimeFD, err := openOrCreateDirAt(workspaceFD, ".apitool")
	if err != nil {
		return nil, fmt.Errorf("open runtime directory: %w", err)
	}
	store := &Store{rootFD: runtimeFD}
	logsFD, err := openOrCreateDirAt(store.rootFD, "logs")
	if err != nil {
		unix.Close(store.rootFD)
		return nil, fmt.Errorf("open runtime logs directory: %w", err)
	}
	unix.Close(logsFD)
	return store, nil
}

func (s *Store) SaveResponse(key Key, response model.Response) error {
	if responseBodySensitive(response.Body) {
		return ErrSensitiveResponse
	}
	dirFD, err := s.responseDir(key)
	if err != nil {
		return err
	}
	defer unix.Close(dirFD)
	data, err := json.Marshal(cachedResponse{CollectionPath: key.CollectionPath, Environment: key.Environment, RequestID: key.RequestID, StatusCode: response.StatusCode, Headers: RedactHeaders(response.Headers), Body: response.Body, Duration: response.Duration, ReceivedAt: response.ReceivedAt})
	if err != nil {
		return fmt.Errorf("encode response cache: %w", err)
	}
	return atomicWriteAt(dirFD, "latest.json", data)
}

func (s *Store) LatestResponse(key Key) (model.Response, error) {
	dirFD, err := s.responseDir(key)
	if err != nil {
		return model.Response{}, err
	}
	defer unix.Close(dirFD)
	data, err := readFileAt(dirFD, "latest.json")
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
	return model.Response{StatusCode: cached.StatusCode, Headers: cached.Headers, Body: cached.Body, Duration: cached.Duration, ReceivedAt: cached.ReceivedAt}, nil
}

func (s *Store) LoadState() (State, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.loadState() }

// SaveState merges values under an advisory workspace lock, avoiding lost
// updates from independently opened Stores.
func (s *Store) SaveState(state State) error {
	state.ensureMaps()
	if err := validateState(state); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.withStateLock(func() error {
		existing, err := s.loadState()
		if err != nil {
			return err
		}
		for collection, environment := range state.LastActiveEnvironment {
			existing.LastActiveEnvironment[collection] = environment
		}
		for panel, preference := range state.PanelPreferences {
			existing.PanelPreferences[panel] = preference
		}
		data, err := json.Marshal(existing)
		if err != nil {
			return fmt.Errorf("encode runtime state: %w", err)
		}
		return atomicWriteAt(s.rootFD, "state.json", data)
	})
}

func (s *Store) loadState() (State, error) {
	data, err := readFileAt(s.rootFD, "state.json")
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

func (s *Store) withStateLock(fn func() error) error {
	fd, err := unix.Openat(s.rootFD, ".state.lock", unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return fmt.Errorf("open state lock: %w", err)
	}
	defer unix.Close(fd)
	if err := unix.Flock(fd, unix.LOCK_EX); err != nil {
		return fmt.Errorf("lock runtime state: %w", err)
	}
	defer unix.Flock(fd, unix.LOCK_UN)
	return fn()
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

func (s *Store) responseDir(key Key) (int, error) {
	segments, err := key.segments()
	if err != nil {
		return -1, err
	}
	dirFD, err := unix.Dup(s.rootFD)
	if err != nil {
		return -1, fmt.Errorf("duplicate runtime directory: %w", err)
	}
	for _, part := range append([]string{"responses"}, segments...) {
		nextFD, err := openOrCreateDirAt(dirFD, part)
		unix.Close(dirFD)
		if err != nil {
			return -1, fmt.Errorf("open response cache path: %w", err)
		}
		dirFD = nextFD
	}
	return dirFD, nil
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

func openOrCreateDirAt(parentFD int, name string) (int, error) {
	if err := unix.Mkdirat(parentFD, name, 0o700); err != nil && !errors.Is(err, unix.EEXIST) {
		return -1, err
	}
	fd, err := unix.Openat(parentFD, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	if err := unix.Fchmod(fd, 0o700); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

func readFileAt(dirFD int, name string) ([]byte, error) {
	fd, err := unix.Openat(dirFD, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			return nil, fs.ErrNotExist
		}
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	defer file.Close()
	return io.ReadAll(file)
}

func atomicWriteAt(dirFD int, name string, data []byte) error {
	temporary := fmt.Sprintf(".write-%d", temporarySequence.Add(1))
	fd, err := unix.Openat(dirFD, temporary, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return fmt.Errorf("create runtime temp file: %w", err)
	}
	defer unix.Unlinkat(dirFD, temporary, 0)
	file := os.NewFile(uintptr(fd), temporary)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write runtime file: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync runtime file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close runtime file: %w", err)
	}
	if err := unix.Renameat(dirFD, temporary, dirFD, name); err != nil {
		return fmt.Errorf("replace runtime file: %w", err)
	}
	if err := syncDirectory(dirFD); err != nil {
		return fmt.Errorf("sync runtime directory: %w", err)
	}
	return nil
}

func (s *Store) appendFile(name string, data []byte) error {
	dirFD, fileName := s.rootFD, name
	if name == filepath.Join("logs", "executions.jsonl") {
		var err error
		dirFD, err = openOrCreateDirAt(s.rootFD, "logs")
		if err != nil {
			return fmt.Errorf("open runtime logs directory: %w", err)
		}
		defer unix.Close(dirFD)
		fileName = "executions.jsonl"
	}
	fd, err := unix.Openat(dirFD, fileName, unix.O_APPEND|unix.O_CREAT|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return fmt.Errorf("open runtime file: %w", err)
	}
	if err := unix.Fchmod(fd, 0o600); err != nil {
		unix.Close(fd)
		return fmt.Errorf("secure runtime file: %w", err)
	}
	file := os.NewFile(uintptr(fd), fileName)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("append runtime file: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync runtime file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close runtime file: %w", err)
	}
	return nil
}

func syncDirectory(fd int) error {
	if err := unix.Fsync(fd); err != nil && !errors.Is(err, unix.EINVAL) && !errors.Is(err, unix.ENOTSUP) {
		return err
	}
	return nil
}
