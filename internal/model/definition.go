package model

// Collection is the metadata and defaults for one API collection.
type Collection struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Auth        *Auth       `yaml:"auth,omitempty"`
	HTTP        *HTTPConfig `yaml:"http,omitempty"`
}

// Environment supplies variables and optional HTTP overrides for a collection.
type Environment struct {
	Name      string            `yaml:"name,omitempty"`
	Variables map[string]string `yaml:"variables,omitempty"`
	HTTP      *HTTPConfig       `yaml:"http,omitempty"`
}

// Group holds optional settings inherited by requests in a directory.
type Group struct {
	Name string `yaml:"name,omitempty"`
	Auth *Auth  `yaml:"auth,omitempty"`
}

type Request struct {
	Name    string        `yaml:"name"`
	Method  string        `yaml:"method"`
	Request RequestConfig `yaml:"request"`
	Auth    *Auth         `yaml:"auth,omitempty"`
}

type RequestConfig struct {
	URL     string            `yaml:"url"`
	Params  map[string]string `yaml:"params,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
	Body    *Body             `yaml:"body,omitempty"`
}

type Body struct {
	Type    string `yaml:"type"`
	Content any    `yaml:"content"`
}

type Auth struct {
	None         bool     `yaml:"-"`
	Type         string   `yaml:"type,omitempty"`
	Grant        string   `yaml:"grant,omitempty"`
	TokenURL     string   `yaml:"token_url,omitempty"`
	ClientID     string   `yaml:"client_id,omitempty"`
	ClientSecret string   `yaml:"client_secret,omitempty"`
	Scopes       []string `yaml:"scopes,omitempty"`
}

type HTTPConfig struct {
	Timeout            string `yaml:"timeout,omitempty"`
	InsecureSkipVerify *bool  `yaml:"insecure_skip_verify,omitempty"`
}
