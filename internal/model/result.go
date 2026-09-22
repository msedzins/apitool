package model

import (
	"net/http"
	"time"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Code     string
	Path     string
	Message  string
	Severity Severity
}

// Response is a completed HTTP response. HTTP status codes are results, not
// execution failures, so this includes 3xx, 4xx, and 5xx responses.
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Duration   time.Duration
	ReceivedAt time.Time
}

type ExecutionStage string

const (
	StageRequestBuild ExecutionStage = "request_build"
	StageOAuth        ExecutionStage = "oauth"
	StageTransport    ExecutionStage = "transport"
)

type ExecutionCategory string

const (
	CategoryRequestBuild ExecutionCategory = "request_build"
	CategoryOAuth        ExecutionCategory = "oauth"
	CategoryDNS          ExecutionCategory = "dns"
	CategoryConnection   ExecutionCategory = "connection"
	CategoryTLS          ExecutionCategory = "tls"
	CategoryCanceled     ExecutionCategory = "canceled"
	CategoryTimeout      ExecutionCategory = "timeout"
)

// ExecutionError contains only safe diagnostic metadata. It deliberately does
// not retain a wrapped network or OAuth error, which may include credentials.
type ExecutionError struct {
	Stage       ExecutionStage
	Category    ExecutionCategory
	SafeMessage string
}

func (e *ExecutionError) Error() string {
	if e == nil {
		return ""
	}
	return e.SafeMessage
}
