package runtime

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"apitool/internal/model"
	"golang.org/x/sys/unix"
)

// HistoryEntry is the safe, searchable record of one execution. It never
// contains request headers, body data, authentication configuration, or an
// execution error message.
type HistoryEntry struct {
	Timestamp      time.Time               `json:"timestamp"`
	CollectionPath string                  `json:"collection_path"`
	Environment    string                  `json:"environment"`
	RequestID      string                  `json:"request_id"`
	Method         string                  `json:"method"`
	Result         string                  `json:"result"`
	StatusCode     int                     `json:"status_code,omitempty"`
	ErrorCategory  model.ExecutionCategory `json:"error_category,omitempty"`
	Duration       time.Duration           `json:"duration_ns"`
}

// AppendHistory appends safe execution metadata. A completed HTTP response has
// a numeric result; an execution error records only its category.
func (s *Store) AppendHistory(key Key, method string, response model.Response, executionError *model.ExecutionError) error {
	if _, err := key.segments(); err != nil {
		return err
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		return fmt.Errorf("history method is required")
	}
	entry := HistoryEntry{
		Timestamp:      time.Now().UTC(),
		CollectionPath: key.CollectionPath,
		Environment:    key.Environment,
		RequestID:      key.RequestID,
		Method:         method,
		Duration:       response.Duration,
	}
	if executionError != nil {
		entry.ErrorCategory = executionError.Category
		entry.Result = string(executionError.Category)
	} else {
		entry.StatusCode = response.StatusCode
		entry.Result = strconv.Itoa(response.StatusCode)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode history entry: %w", err)
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendFile("history.jsonl", data)
}

// SearchHistory returns entries whose collection, request, method, or result
// contains query, ignoring case. Corrupt individual JSONL lines are skipped.
func (s *Store) SearchHistory(query string) ([]HistoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fd, err := unix.Openat(s.rootFD, "history.jsonl", unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return []HistoryEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open history: %w", err)
	}
	file := os.NewFile(uintptr(fd), "history.jsonl")
	defer file.Close()

	needle := strings.ToLower(strings.TrimSpace(query))
	entries := make([]HistoryEntry, 0)
	reader := bufio.NewReaderSize(file, 64*1024)
	for {
		line, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			if err := discardHistoryLine(reader); err != nil && !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("read history: %w", err)
			}
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}
		if errors.Is(err, io.EOF) {
			break // A partial final write is ignored as one malformed record.
		}
		if err != nil {
			return nil, fmt.Errorf("read history: %w", err)
		}
		var entry HistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil || !validHistoryEntry(entry) {
			continue
		}
		if historyMatches(entry, needle) {
			entries = append(entries, entry)
		}
	}
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
	return entries, nil
}

func discardHistoryLine(reader *bufio.Reader) error {
	for {
		_, err := reader.ReadSlice('\n')
		if !errors.Is(err, bufio.ErrBufferFull) {
			return err
		}
	}
}

func validHistoryEntry(entry HistoryEntry) bool {
	if entry.Timestamp.IsZero() || strings.TrimSpace(entry.Method) == "" || entry.Result == "" {
		return false
	}
	if _, err := (Key{CollectionPath: entry.CollectionPath, Environment: entry.Environment, RequestID: entry.RequestID}).segments(); err != nil {
		return false
	}
	if entry.ErrorCategory != "" {
		return entry.StatusCode == 0 && entry.Result == string(entry.ErrorCategory)
	}
	return entry.StatusCode >= 100 && entry.StatusCode <= 599 && entry.Result == strconv.Itoa(entry.StatusCode)
}

func historyMatches(entry HistoryEntry, needle string) bool {
	if needle == "" {
		return true
	}
	for _, value := range []string{entry.CollectionPath, entry.RequestID, entry.Method, entry.Result, strconv.Itoa(entry.StatusCode), string(entry.ErrorCategory)} {
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	return false
}
