// Package collection loads and saves YAML API definition files.
package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"apitool/internal/model"

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
	if err := rejectLiteralSecret(collection.Auth); err != nil {
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
	var document requestDocument
	if err := loadYAML(path, &document); err != nil {
		return model.Request{}, err
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
		Request: document.Request,
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
	if err := rejectLiteralSecret(auth); err != nil {
		return model.Group{}, err
	}
	return model.Group{Name: document.Name, Auth: auth}, nil
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
	Name    string              `yaml:"name"`
	Method  string              `yaml:"method"`
	Request model.RequestConfig `yaml:"request"`
	Auth    yaml.Node           `yaml:"auth,omitempty"`
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
