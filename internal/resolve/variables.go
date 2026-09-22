// Package resolve derives executable request settings from collection definitions.
package resolve

import (
	"os"
	"regexp"

	"apitool/internal/model"
)

var (
	environmentVariablePattern = regexp.MustCompile(`\{\{([^{}]+)\}\}`)
	processVariablePattern     = regexp.MustCompile(`\$\{([^{}]+)\}`)

	// LookupEnv supplies process environment values. Tests and embedders may replace
	// it to make resolution deterministic.
	LookupEnv = os.LookupEnv
)

func resolveString(value, path string, variables map[string]string, lookup func(string) (string, bool)) (string, []model.Diagnostic) {
	var diagnostics []model.Diagnostic
	value = environmentVariablePattern.ReplaceAllStringFunc(value, func(match string) string {
		name := environmentVariablePattern.FindStringSubmatch(match)[1]
		resolved, ok := variables[name]
		if !ok {
			diagnostics = append(diagnostics, missingVariable(path, name))
			return match
		}
		return resolved
	})
	value = processVariablePattern.ReplaceAllStringFunc(value, func(match string) string {
		name := processVariablePattern.FindStringSubmatch(match)[1]
		resolved, ok := lookup(name)
		if !ok {
			diagnostics = append(diagnostics, missingVariable(path, name))
			return match
		}
		return resolved
	})
	return value, diagnostics
}

func missingVariable(path, name string) model.Diagnostic {
	return model.Diagnostic{
		Code:     "variable_missing",
		Path:     path,
		Message:  "variable " + name + " is not defined",
		Severity: model.SeverityError,
	}
}
