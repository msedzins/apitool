package model

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
