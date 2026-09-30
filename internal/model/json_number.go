package model

import (
	"fmt"
	"regexp"

	"gopkg.in/yaml.v3"
)

// JSONNumber retains a JSON numeric token without converting it through float64.
type JSONNumber string

var jsonNumberPattern = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][+-]?[0-9]+)?$`)

func (n JSONNumber) MarshalJSON() ([]byte, error) {
	if !jsonNumberPattern.MatchString(string(n)) {
		return nil, fmt.Errorf("invalid JSON number %q", n)
	}
	return []byte(n), nil
}

func (n JSONNumber) MarshalYAML() (any, error) {
	if !jsonNumberPattern.MatchString(string(n)) {
		return nil, fmt.Errorf("invalid JSON number %q", n)
	}
	tag := "!!int"
	for _, r := range n {
		if r == '.' || r == 'e' || r == 'E' {
			tag = "!!float"
			break
		}
	}
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: string(n)}, nil
}
