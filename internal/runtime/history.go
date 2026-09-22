package runtime

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"apitool/internal/model"
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
	file, err := os.Open(filepath.Join(s.root, "history.jsonl"))
	if os.IsNotExist(err) {
		return []HistoryEntry{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open history: %w", err)
	}
	defer file.Close()

	needle := strings.ToLower(strings.TrimSpace(query))
	entries := make([]HistoryEntry, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		var entry HistoryEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}
		if historyMatches(entry, needle) {
			entries = append(entries, entry)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	return entries, nil
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
