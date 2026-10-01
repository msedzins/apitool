package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"apitool/internal/model"
)

type jsonScalar struct {
	path  []string
	label string
	value any
}

func jsonScalars(text string) ([]jsonScalar, error) {
	value, err := decodeJSONValue(text)
	if err != nil {
		return nil, err
	}
	var scalars []jsonScalar
	var walk func(any, []string)
	walk = func(value any, path []string) {
		switch node := value.(type) {
		case map[string]any:
			keys := make([]string, 0, len(node))
			for key := range node {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				walk(node[key], appendPath(path, "."+key))
			}
		case []any:
			for i, child := range node {
				walk(child, appendPath(path, "["+strconv.Itoa(i)+"]"))
			}
		default:
			label := "$"
			if len(path) > 0 {
				label = "$" + strings.Join(path, "")
			}
			scalars = append(scalars, jsonScalar{path: append([]string(nil), path...), label: label, value: node})
		}
	}
	walk(value, nil)
	return scalars, nil
}

func appendPath(path []string, key string) []string {
	result := append([]string(nil), path...)
	result = append(result, key)
	return result
}

func setJSONScalar(text string, path []string, replacement string) (string, error) {
	value, err := decodeJSONValue(text)
	if err != nil {
		return "", err
	}
	if len(path) == 0 {
		updated, err := parseJSONScalar(replacement, value)
		if err != nil {
			return "", err
		}
		value = updated
	} else {
		parent := value
		for _, part := range path[:len(path)-1] {
			if strings.HasPrefix(part, "[") {
				index, _ := strconv.Atoi(strings.Trim(part, "[]"))
				parent = parent.([]any)[index]
			} else {
				parent = parent.(map[string]any)[strings.TrimPrefix(part, ".")]
			}
		}
		last := path[len(path)-1]
		var old any
		if strings.HasPrefix(last, "[") {
			index, _ := strconv.Atoi(strings.Trim(last, "[]"))
			old = parent.([]any)[index]
		} else {
			old = parent.(map[string]any)[strings.TrimPrefix(last, ".")]
		}
		updated, err := parseJSONScalar(replacement, old)
		if err != nil {
			return "", err
		}
		if strings.HasPrefix(last, "[") {
			index, _ := strconv.Atoi(strings.Trim(last, "[]"))
			parent.([]any)[index] = updated
		} else {
			parent.(map[string]any)[strings.TrimPrefix(last, ".")] = updated
		}
	}
	data, err := json.MarshalIndent(value, "", "  ")
	return string(data), err
}

func parseJSONScalar(text string, old any) (any, error) {
	switch old.(type) {
	case string:
		return text, nil
	case json.Number, model.JSONNumber:
		var number json.Number
		decoder := json.NewDecoder(strings.NewReader(text))
		decoder.UseNumber()
		if err := decoder.Decode(&number); err != nil {
			return nil, fmt.Errorf("invalid number: %w", err)
		}
		return model.JSONNumber(number.String()), nil
	case bool:
		return strconv.ParseBool(text)
	case nil:
		if text == "null" {
			return nil, nil
		}
	}
	return nil, fmt.Errorf("unsupported JSON scalar")
}

func formatJSONScalar(value any) string {
	data, _ := json.Marshal(value)
	return string(bytes.TrimSpace(data))
}
