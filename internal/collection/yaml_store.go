// Package collection loads and saves YAML API definition files.
package collection

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"apitool/internal/model"
	"apitool/internal/validate"

	"gopkg.in/yaml.v3"
)

// LoadCollectionMeta reads collection metadata and its HTTP defaults.
func LoadCollectionMeta(path string) (model.Collection, error) {
	var document collectionDocument
	if err := loadYAML(path, &document); err != nil {
		return model.Collection{}, err
	}
	if strings.TrimSpace(document.Name) == "" {
		return model.Collection{}, fmt.Errorf("collection name is required")
	}
	auth, err := decodeAuth(document.Auth)
	if err != nil {
		return model.Collection{}, fmt.Errorf("decode auth: %w", err)
	}
	collection := model.Collection{
		Name: document.Name, Description: document.Description, Auth: auth, HTTP: document.HTTP,
	}
	if err := validateHTTPConfig(collection.HTTP); err != nil {
		return model.Collection{}, fmt.Errorf("collection HTTP configuration: %w", err)
	}
	if err := validateAuthSource(path, collection.Auth); err != nil {
		return model.Collection{}, err
	}
	return collection, nil
}

// LoadEnvironment reads an environment definition and its HTTP overrides.
func LoadEnvironment(path string) (model.Environment, error) {
	var environment model.Environment
	if err := loadYAML(path, &environment); err != nil {
		return model.Environment{}, err
	}
	key := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if environment.Name != "" && environment.Name != key {
		return model.Environment{}, fmt.Errorf("environment name %q must match filename key %q", environment.Name, key)
	}
	if err := validateHTTPConfig(environment.HTTP); err != nil {
		return model.Environment{}, fmt.Errorf("environment HTTP configuration: %w", err)
	}
	return environment, nil
}

// LoadRequest reads a request, accepting auth: none and either supported scope form.
func LoadRequest(path string) (model.Request, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Request{}, err
	}
	var document requestDocument
	if err := yaml.Unmarshal(data, &document); err != nil {
		return model.Request{}, fmt.Errorf("decode YAML: %w", err)
	}
	requestConfig, err := decodeRequestConfig(document.Request)
	if err != nil {
		return model.Request{}, fmt.Errorf("decode request body: %w", err)
	}

	auth, err := decodeAuth(document.Auth)
	if err != nil {
		return model.Request{}, fmt.Errorf("decode auth: %w", err)
	}
	if err := rejectLiteralSecret(auth); err != nil {
		return model.Request{}, err
	}

	return model.Request{
		Name:    document.Name,
		Method:  strings.ToUpper(document.Method),
		Request: requestConfig,
		Auth:    auth,
	}, nil
}

// LoadGroup reads optional group settings inherited by requests in its directory.
func LoadGroup(path string) (model.Group, error) {
	var document groupDocument
	if err := loadYAML(path, &document); err != nil {
		return model.Group{}, err
	}

	auth, err := decodeAuth(document.Auth)
	if err != nil {
		return model.Group{}, fmt.Errorf("decode auth: %w", err)
	}
	if err := validateAuthSource(path, auth); err != nil {
		return model.Group{}, err
	}
	return model.Group{Name: document.Name, Auth: auth}, nil
}

func validateAuthSource(path string, auth *model.Auth) error {
	diagnostics := validate.Auth(auth)
	if len(diagnostics) == 0 {
		return nil
	}
	items := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		items = append(items, diagnostic.Path+": "+diagnostic.Message)
	}
	return fmt.Errorf("%s: %s", path, strings.Join(items, "; "))
}

// SaveRequest writes request to path atomically. Client secrets must remain
// process environment references, never literal values.
func SaveRequest(path string, request model.Request) error {
	if err := rejectLiteralSecret(request.Auth); err != nil {
		return err
	}

	document := requestWriteDocument{
		Name:    request.Name,
		Method:  strings.ToUpper(request.Method),
		Request: request.Request,
	}
	if request.Auth != nil {
		if request.Auth.None {
			document.Auth = "none"
		} else {
			document.Auth = request.Auth
		}
	}

	data, err := yaml.Marshal(document)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	return writeAtomic(path, data)
}

type requestDocument struct {
	Name    string                `yaml:"name"`
	Method  string                `yaml:"method"`
	Request requestConfigDocument `yaml:"request"`
	Auth    yaml.Node             `yaml:"auth,omitempty"`
}
type requestConfigDocument struct {
	URL     string            `yaml:"url"`
	Params  map[string]string `yaml:"params,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    *bodyDocument     `yaml:"body,omitempty"`
}
type bodyDocument struct {
	Type    string    `yaml:"type"`
	Content yaml.Node `yaml:"content"`
}

func decodeRequestConfig(document requestConfigDocument) (model.RequestConfig, error) {
	config := model.RequestConfig{URL: document.URL, Params: document.Params, Headers: document.Headers}
	if document.Body == nil {
		return config, nil
	}
	content, err := decodeBodyNode(&document.Body.Content)
	if err != nil {
		return model.RequestConfig{}, err
	}
	config.Body = &model.Body{Type: document.Body.Type, Content: content}
	return config, nil
}
func decodeBodyNode(node *yaml.Node) (any, error) {
	return decodeBodyNodeVisited(node, map[*yaml.Node]bool{})
}
func decodeBodyNodeVisited(node *yaml.Node, stack map[*yaml.Node]bool) (any, error) {
	if node == nil || node.Kind == 0 {
		return nil, nil
	}
	if stack[node] {
		return nil, fmt.Errorf("cyclic YAML alias in JSON body")
	}
	stack[node] = true
	defer delete(stack, node)
	if node.Kind == yaml.AliasNode {
		return decodeBodyNodeVisited(node.Alias, stack)
	}
	switch node.Kind {
	case yaml.MappingNode:
		result := make(map[string]any, len(node.Content)/2)
		var mergeNodes []*yaml.Node
		seenKeys := map[string]bool{}
		for index := 0; index < len(node.Content); index += 2 {
			key, value := node.Content[index], node.Content[index+1]
			if key.Tag == "!!merge" {
				mergeNodes = append(mergeNodes, value)
				continue
			}
			if key.Tag != "!!str" {
				return nil, fmt.Errorf("JSON body object keys must be strings")
			}
			if seenKeys[key.Value] {
				return nil, fmt.Errorf("duplicate JSON body key %q", key.Value)
			}
			seenKeys[key.Value] = true
		}
		for _, mergeNode := range mergeNodes {
			merged, err := decodeBodyNodeVisited(mergeNode, stack)
			if err != nil {
				return nil, err
			}
			maps := []map[string]any{}
			switch value := merged.(type) {
			case map[string]any:
				maps = append(maps, value)
			case []any:
				for _, item := range value {
					mapping, ok := item.(map[string]any)
					if !ok {
						return nil, fmt.Errorf("YAML merge sequence must contain mappings")
					}
					maps = append(maps, mapping)
				}
			default:
				return nil, fmt.Errorf("YAML merge value must be a mapping or sequence of mappings")
			}
			for _, mapping := range maps {
				for key, value := range mapping {
					if _, exists := result[key]; !exists {
						result[key] = value
					}
				}
			}
		}
		for index := 0; index < len(node.Content); index += 2 {
			key, valueNode := node.Content[index], node.Content[index+1]
			if key.Tag == "!!merge" {
				continue
			}
			value, err := decodeBodyNodeVisited(valueNode, stack)
			if err != nil {
				return nil, err
			}
			result[key.Value] = value
		}
		return result, nil
	case yaml.SequenceNode:
		result := make([]any, len(node.Content))
		for index, child := range node.Content {
			value, err := decodeBodyNodeVisited(child, stack)
			if err != nil {
				return nil, err
			}
			result[index] = value
		}
		return result, nil
	case yaml.ScalarNode:
		switch node.Tag {
		case "!!str":
			return node.Value, nil
		case "!!null":
			return nil, nil
		case "!!bool":
			var value bool
			if err := node.Decode(&value); err != nil {
				return nil, err
			}
			return value, nil
		case "!!int":
			return decodeYAMLInteger(node.Value)
		case "!!float":
			return decodeYAMLFloat(node.Value)
		case "!!timestamp":
			var value time.Time
			if err := node.Decode(&value); err != nil {
				return nil, err
			}
			return value, nil
		case "!!binary":
			var value any
			if err := node.Decode(&value); err != nil {
				return nil, err
			}
			return value, nil
		default:
			return nil, fmt.Errorf("unsupported YAML scalar %s in JSON body", node.Tag)
		}
	default:
		return nil, fmt.Errorf("unsupported YAML node in JSON body")
	}
}
func decodeYAMLInteger(raw string) (any, error) {
	value := strings.ReplaceAll(raw, "_", "")
	sign := 1
	if strings.HasPrefix(value, "-") {
		sign = -1
		value = value[1:]
	} else if strings.HasPrefix(value, "+") {
		value = value[1:]
	}
	base := 10
	switch {
	case strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X"):
		base, value = 16, value[2:]
	case strings.HasPrefix(value, "0o") || strings.HasPrefix(value, "0O"):
		base, value = 8, value[2:]
	case strings.HasPrefix(value, "0b") || strings.HasPrefix(value, "0B"):
		base, value = 2, value[2:]
	case len(value) > 1 && value[0] == '0':
		base, value = 8, value[1:]
	}
	number, ok := new(big.Int).SetString(value, base)
	if !ok {
		return nil, fmt.Errorf("invalid YAML integer %q", raw)
	}
	if sign < 0 {
		number.Neg(number)
	}
	decimal := number.String()
	if integer, err := strconv.ParseInt(decimal, 10, 64); err == nil {
		return int(integer), nil
	}
	return model.JSONNumber(decimal), nil
}
func decodeYAMLFloat(raw string) (any, error) {
	value := strings.ReplaceAll(raw, "_", "")
	sign := ""
	if strings.HasPrefix(value, "+") {
		value = value[1:]
	} else if strings.HasPrefix(value, "-") {
		sign, value = "-", value[1:]
	}
	exponent := ""
	if index := strings.IndexAny(value, "eE"); index >= 0 {
		exponent, value = value[index:], value[:index]
	}
	before, after, hasDot := strings.Cut(value, ".")
	if !hasDot {
		after = ""
	}
	if before == "" {
		before = "0"
	}
	trimmed := strings.TrimLeft(before, "0")
	if trimmed != "" {
		before = trimmed
	} else {
		before = "0"
	}
	if hasDot && after == "" {
		after = "0"
	}
	canonical := sign + before
	if hasDot {
		canonical += "." + after
	}
	canonical += exponent
	number := model.JSONNumber(canonical)
	if _, err := json.Marshal(number); err != nil {
		return nil, fmt.Errorf("unsupported YAML float %q in JSON body", raw)
	}
	return number, nil
}

type collectionDocument struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	Auth        yaml.Node         `yaml:"auth,omitempty"`
	HTTP        *model.HTTPConfig `yaml:"http,omitempty"`
}

type groupDocument struct {
	Name string    `yaml:"name,omitempty"`
	Auth yaml.Node `yaml:"auth,omitempty"`
}

type requestWriteDocument struct {
	Name    string              `yaml:"name"`
	Method  string              `yaml:"method"`
	Request model.RequestConfig `yaml:"request"`
	Auth    any                 `yaml:"auth,omitempty"`
}

type authDocument struct {
	Type         string    `yaml:"type,omitempty"`
	Grant        string    `yaml:"grant,omitempty"`
	TokenURL     string    `yaml:"token_url,omitempty"`
	ClientID     string    `yaml:"client_id,omitempty"`
	ClientSecret string    `yaml:"client_secret,omitempty"`
	Scopes       yaml.Node `yaml:"scopes,omitempty"`
}

func decodeAuth(node yaml.Node) (*model.Auth, error) {
	if node.Kind == 0 {
		return nil, nil
	}
	if node.Kind == yaml.ScalarNode {
		if node.Tag == "!!str" && node.Value == "none" {
			return &model.Auth{None: true}, nil
		}
		return nil, fmt.Errorf("auth must be a mapping or none")
	}
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("auth must be a mapping or none")
	}

	var document authDocument
	if err := node.Decode(&document); err != nil {
		return nil, err
	}
	scopes, err := decodeScopes(document.Scopes)
	if err != nil {
		return nil, err
	}
	return &model.Auth{
		Type: document.Type, Grant: document.Grant, TokenURL: document.TokenURL,
		ClientID: document.ClientID, ClientSecret: document.ClientSecret, Scopes: scopes,
	}, nil
}

func decodeScopes(node yaml.Node) ([]string, error) {
	if node.Kind == 0 {
		return nil, nil
	}
	switch node.Kind {
	case yaml.SequenceNode:
		var scopes []string
		if err := node.Decode(&scopes); err != nil {
			return nil, fmt.Errorf("decode scopes: %w", err)
		}
		return scopes, nil
	case yaml.ScalarNode:
		if node.Tag != "!!str" {
			return nil, fmt.Errorf("scopes must be a sequence or space-delimited string")
		}
		return strings.Fields(node.Value), nil
	default:
		return nil, fmt.Errorf("scopes must be a sequence or space-delimited string")
	}
}

func loadYAML(path string, destination any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("decode YAML: %w", err)
	}
	return nil
}

func rejectLiteralSecret(auth *model.Auth) error {
	if auth == nil || auth.None || auth.ClientSecret == "" {
		return nil
	}
	if isEnvironmentReference(auth.ClientSecret) {
		return nil
	}
	return fmt.Errorf("client_secret must be a single process environment reference")
}

func isEnvironmentReference(value string) bool {
	if len(value) < 4 || !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return false
	}
	for index, r := range value[2 : len(value)-1] {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (index > 0 && r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

func validateHTTPConfig(config *model.HTTPConfig) error {
	if config == nil || config.Timeout == "" {
		return nil
	}
	duration, err := time.ParseDuration(config.Timeout)
	if err != nil || duration <= 0 {
		return fmt.Errorf("timeout must be a positive Go duration")
	}
	return nil
}

func writeAtomic(path string, data []byte) error {
	directory := filepath.Dir(path)
	info, err := os.Stat(path)
	mode := os.FileMode(0o644)
	if err == nil {
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}

	temporary, err := os.CreateTemp(directory, ".apitool-*.yaml")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}
