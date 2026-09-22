package resolve

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"apitool/internal/model"
)

// Effective resolves environment and process placeholders, then applies HTTP and
// auth inheritance for one request. Environment values are expanded once before
// process values; replacement values are never recursively re-expanded.
func Effective(collection model.Collection, environment model.Environment, groups []model.Group, request model.Request) (model.EffectiveRequest, []model.Diagnostic) {
	lookup := LookupEnv
	if lookup == nil {
		lookup = func(string) (string, bool) { return "", false }
	}

	var diagnostics []model.Diagnostic
	resolve := func(value, path string) string {
		resolved, found := resolveString(value, path, environment.Variables, lookup)
		diagnostics = append(diagnostics, found...)
		return resolved
	}

	effective := model.EffectiveRequest{
		Method:  strings.ToUpper(request.Method),
		URL:     resolve(request.Request.URL, "request.url"),
		Params:  resolveMap(request.Request.Params, "request.params", resolve),
		Headers: resolveMap(request.Request.Headers, "request.headers", resolve),
		Auth:    resolveAuth(selectAuth(collection.Auth, groups, request.Auth), resolve),
	}

	effective.Body = resolveBody(request.Request.Body, resolve, &diagnostics)
	effective.Timeout, effective.InsecureSkipVerify = resolveHTTP(collection.HTTP, environment.HTTP, &diagnostics)
	return effective, diagnostics
}

func resolveMap(values map[string]string, path string, resolve func(string, string) string) map[string]string {
	if values == nil {
		return nil
	}
	resolved := make(map[string]string, len(values))
	for _, key := range sortedKeys(values) {
		value := values[key]
		resolved[key] = resolve(value, path+"."+key)
	}
	return resolved
}

func resolveBody(body *model.Body, resolve func(string, string) string, diagnostics *[]model.Diagnostic) []byte {
	if body == nil {
		return nil
	}
	path := "request.body.content"
	switch body.Type {
	case "raw":
		value, ok := body.Content.(string)
		if !ok {
			*diagnostics = append(*diagnostics, invalidBody(path))
			return nil
		}
		return []byte(resolve(value, path))
	case "json":
		content := resolveJSON(body.Content, path, resolve)
		encoded, err := json.Marshal(content)
		if err != nil {
			*diagnostics = append(*diagnostics, invalidBody(path))
			return nil
		}
		return encoded
	default:
		*diagnostics = append(*diagnostics, invalidBody(path))
		return nil
	}
}

func resolveJSON(value any, path string, resolve func(string, string) string) any {
	switch value := value.(type) {
	case string:
		return resolve(value, path)
	case []any:
		resolved := make([]any, len(value))
		for index, item := range value {
			resolved[index] = resolveJSON(item, path+"["+strconv.Itoa(index)+"]", resolve)
		}
		return resolved
	case map[string]any:
		resolved := make(map[string]any, len(value))
		for _, key := range sortedKeys(value) {
			item := value[key]
			resolved[key] = resolveJSON(item, path+"."+key, resolve)
		}
		return resolved
	case []string:
		resolved := make([]string, len(value))
		for index, item := range value {
			resolved[index] = resolve(item, path+"["+strconv.Itoa(index)+"]")
		}
		return resolved
	case map[string]string:
		resolved := make(map[string]string, len(value))
		for _, key := range sortedKeys(value) {
			item := value[key]
			resolved[key] = resolve(item, path+"."+key)
		}
		return resolved
	default:
		return value
	}
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func invalidBody(path string) model.Diagnostic {
	return model.Diagnostic{
		Code:     "body_json_invalid",
		Path:     path,
		Message:  "JSON body content must be JSON-serializable",
		Severity: model.SeverityError,
	}
}

func selectAuth(collection *model.Auth, groups []model.Group, request *model.Auth) *model.Auth {
	effective := collection
	for _, group := range groups {
		if group.Auth != nil {
			effective = group.Auth
		}
	}
	if request != nil {
		effective = request
	}
	return effective
}

func resolveAuth(auth *model.Auth, resolve func(string, string) string) *model.Auth {
	if auth == nil {
		return nil
	}
	resolved := *auth
	resolved.TokenURL = resolve(auth.TokenURL, "auth.token_url")
	resolved.ClientID = resolve(auth.ClientID, "auth.client_id")
	resolved.ClientSecret = resolve(auth.ClientSecret, "auth.client_secret")
	if auth.Scopes != nil {
		resolved.Scopes = make([]string, len(auth.Scopes))
		for index, scope := range auth.Scopes {
			resolved.Scopes[index] = resolve(scope, "auth.scopes["+strconv.Itoa(index)+"]")
		}
	}
	return &resolved
}

func resolveHTTP(collection, environment *model.HTTPConfig, diagnostics *[]model.Diagnostic) (time.Duration, bool) {
	var timeout string
	var insecureSkipVerify *bool
	if collection != nil {
		timeout = collection.Timeout
		insecureSkipVerify = collection.InsecureSkipVerify
	}
	if environment != nil {
		if environment.Timeout != "" {
			timeout = environment.Timeout
		}
		if environment.InsecureSkipVerify != nil {
			insecureSkipVerify = environment.InsecureSkipVerify
		}
	}

	if timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil || parsed <= 0 {
			*diagnostics = append(*diagnostics, model.Diagnostic{
				Code: "timeout_invalid", Path: "http.timeout", Message: "timeout must be a positive Go duration", Severity: model.SeverityError,
			})
		} else {
			return parsed, insecureSkipVerify != nil && *insecureSkipVerify
		}
	}
	return 0, insecureSkipVerify != nil && *insecureSkipVerify
}
