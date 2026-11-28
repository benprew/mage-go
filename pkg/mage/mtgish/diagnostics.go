package mtgish

import (
	"errors"
	"fmt"
	"strings"
)

// DiagnosticCode identifies the reason that an import failed.
type DiagnosticCode string

const (
	InvalidInput         DiagnosticCode = "invalid_input"
	UnsupportedConstruct DiagnosticCode = "unsupported_construct"
	EngineGap            DiagnosticCode = "engine_gap"
	RegistrationConflict DiagnosticCode = "registration_conflict"
	ResourceLimit        DiagnosticCode = "resource_limit"
	IOError              DiagnosticCode = "io_error"
)

// CompileError identifies an import blocker and its source location.
// Record is a physical line number that starts at one. Zero means no location.
// Path is a JSON Pointer. An empty path identifies the root value.
type CompileError struct {
	Source  string
	Record  int
	Card    string
	Path    string
	Code    DiagnosticCode
	Family  string
	Tag     string
	Message string
}

func (e *CompileError) Error() string {
	var location strings.Builder
	if e.Source != "" {
		location.WriteByte(' ')
		location.WriteString(e.Source)
	}
	if e.Record != 0 {
		fmt.Fprintf(&location, ":%d", e.Record)
	}
	if e.Card != "" {
		fmt.Fprintf(&location, " card %q", e.Card)
	}
	if e.Path != "" {
		fmt.Fprintf(&location, " at %s", e.Path)
	}
	return fmt.Sprintf("mtgish%s: %s: %s", location.String(), e.Code, e.Message)
}

func diagnostic(path string, code DiagnosticCode, family, tag, message string) *CompileError {
	return &CompileError{Path: path, Code: code, Family: family, Tag: tag, Message: message}
}

func withCard(err error, name string) error {
	if source, ok := errors.AsType[*CompileError](err); ok {
		result := *source
		result.Card = name
		return &result
	}
	return &CompileError{Card: name, Code: InvalidInput, Message: err.Error()}
}
