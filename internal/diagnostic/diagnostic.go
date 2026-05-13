// Package diagnostic provides structured error and warning types used across GOFVML.
package diagnostic

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Severity indicates the severity of a diagnostic.
type Severity int

const (
	// SeverityInfo is an informational diagnostic.
	SeverityInfo Severity = iota
	// SeverityWarning indicates a non-fatal issue or partial success condition.
	SeverityWarning
	// SeverityError indicates a fatal or blocking issue.
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return fmt.Sprintf("severity(%d)", s)
	}
}

// Category classifies the domain of a diagnostic.
type Category int

const (
	// CategorySource indicates a physical memory source error (e.g., /dev/mem unavailable).
	CategorySource Category = iota
	// CategoryPolicy indicates a privilege or security policy error (e.g., CAP_SYS_ADMIN missing).
	CategoryPolicy
	// CategoryParse indicates a parsing error (e.g., malformed /proc/iomem line).
	CategoryParse
	// CategoryFormat indicates an image format error (e.g., invalid LiME header).
	CategoryFormat
	// CategoryUpload indicates an upload error (e.g., HTTP PUT rejected).
	CategoryUpload
	// CategoryProcess indicates a process acquisition error (e.g., PID not found).
	CategoryProcess
	// CategoryInternal indicates an unexpected internal error.
	CategoryInternal
)

func (c Category) String() string {
	switch c {
	case CategorySource:
		return "source"
	case CategoryPolicy:
		return "policy"
	case CategoryParse:
		return "parse"
	case CategoryFormat:
		return "format"
	case CategoryUpload:
		return "upload"
	case CategoryProcess:
		return "process"
	case CategoryInternal:
		return "internal"
	default:
		return fmt.Sprintf("category(%d)", c)
	}
}

// Diagnostic is a structured diagnostic message with machine-readable
// categorization and human-readable explanation.
type Diagnostic struct {
	// Severity indicates whether this is info, warning, or error.
	Severity Severity
	// Category classifies the diagnostic domain.
	Category Category
	// Operation is the high-level operation that produced this diagnostic
	// (e.g., "physical acquisition", "process dump", "conversion").
	Operation string
	// Message is the human-readable description.
	Message string
	// Target identifies the specific target of the diagnostic
	// (e.g., source path, PID, file path).
	Target string
	// Suggestion is an optional remediation hint for operators.
	Suggestion string
	// Cause is the underlying error, if any.
	Cause error
}

// Error implements the error interface. It returns a formatted string
// that preserves the severity, category, and message.
func (d *Diagnostic) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "[%s:%s]", d.Severity, d.Category)
	if d.Operation != "" {
		fmt.Fprintf(&b, " %s:", d.Operation)
	}
	fmt.Fprintf(&b, " %s", d.Message)
	if d.Target != "" {
		fmt.Fprintf(&b, " (target: %s)", d.Target)
	}
	if d.Suggestion != "" {
		fmt.Fprintf(&b, "; suggestion: %s", d.Suggestion)
	}
	if d.Cause != nil {
		fmt.Fprintf(&b, ": %v", d.Cause)
	}
	return b.String()
}

// Is reports whether this diagnostic matches target for errors.Is.
func (d *Diagnostic) Is(target error) bool {
	if target == nil {
		return d == nil
	}
	if t, ok := target.(*Diagnostic); ok {
		return d.Severity == t.Severity && d.Category == t.Category
	}
	return false
}

// Unwrap returns the underlying cause.
func (d *Diagnostic) Unwrap() error {
	return d.Cause
}

// New creates a new Diagnostic.
func New(severity Severity, category Category, message string) *Diagnostic {
	return &Diagnostic{
		Severity: severity,
		Category: category,
		Message:  message,
	}
}

// WithOperation sets the operation field.
func (d *Diagnostic) WithOperation(op string) *Diagnostic {
	d.Operation = op
	return d
}

// WithTarget sets the target field.
func (d *Diagnostic) WithTarget(target string) *Diagnostic {
	d.Target = target
	return d
}

// WithSuggestion sets the suggestion field.
func (d *Diagnostic) WithSuggestion(s string) *Diagnostic {
	d.Suggestion = s
	return d
}

// WithCause sets the cause field.
func (d *Diagnostic) WithCause(err error) *Diagnostic {
	d.Cause = err
	return d
}

// jsonDiagnostic is the JSON representation of a Diagnostic.
type jsonDiagnostic struct {
	Severity   string `json:"severity"`
	Category   string `json:"category"`
	Operation  string `json:"operation,omitempty"`
	Message    string `json:"message"`
	Target     string `json:"target,omitempty"`
	Suggestion string `json:"suggestion,omitempty"`
	Cause      string `json:"cause,omitempty"`
}

// MarshalJSON implements json.Marshaler.
func (d *Diagnostic) MarshalJSON() ([]byte, error) {
	j := jsonDiagnostic{
		Severity:   d.Severity.String(),
		Category:   d.Category.String(),
		Operation:  d.Operation,
		Message:    d.Message,
		Target:     d.Target,
		Suggestion: d.Suggestion,
	}
	if d.Cause != nil {
		j.Cause = d.Cause.Error()
	}
	return json.Marshal(j)
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Diagnostic) UnmarshalJSON(data []byte) error {
	var j jsonDiagnostic
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}

	switch j.Severity {
	case "info":
		d.Severity = SeverityInfo
	case "warning":
		d.Severity = SeverityWarning
	case "error":
		d.Severity = SeverityError
	default:
		return fmt.Errorf("unknown severity: %q", j.Severity)
	}

	switch j.Category {
	case "source":
		d.Category = CategorySource
	case "policy":
		d.Category = CategoryPolicy
	case "parse":
		d.Category = CategoryParse
	case "format":
		d.Category = CategoryFormat
	case "upload":
		d.Category = CategoryUpload
	case "process":
		d.Category = CategoryProcess
	case "internal":
		d.Category = CategoryInternal
	default:
		return fmt.Errorf("unknown category: %q", j.Category)
	}

	d.Operation = j.Operation
	d.Message = j.Message
	d.Target = j.Target
	d.Suggestion = j.Suggestion
	if j.Cause != "" {
		d.Cause = errors.New(j.Cause)
	}
	return nil
}

// Common factory functions for convenience.

// SourceError creates a source-category error diagnostic.
func SourceError(message string) *Diagnostic {
	return New(SeverityError, CategorySource, message)
}

// PolicyError creates a policy-category error diagnostic.
func PolicyError(message string) *Diagnostic {
	return New(SeverityError, CategoryPolicy, message)
}

// ParseError creates a parse-category error diagnostic.
func ParseError(message string) *Diagnostic {
	return New(SeverityError, CategoryParse, message)
}

// FormatError creates a format-category error diagnostic.
func FormatError(message string) *Diagnostic {
	return New(SeverityError, CategoryFormat, message)
}

// UploadError creates an upload-category error diagnostic.
func UploadError(message string) *Diagnostic {
	return New(SeverityError, CategoryUpload, message)
}

// ProcessError creates a process-category error diagnostic.
func ProcessError(message string) *Diagnostic {
	return New(SeverityError, CategoryProcess, message)
}

// Warning creates a warning diagnostic with the given category.
func Warning(category Category, message string) *Diagnostic {
	return New(SeverityWarning, category, message)
}

// Info creates an info diagnostic with the given category.
func Info(category Category, message string) *Diagnostic {
	return New(SeverityInfo, category, message)
}

// Result holds the outcome of an operation, including any warnings
// and operational details.
type Result struct {
	// Success is true if the operation completed without fatal errors.
	// Partial success (with warnings) still sets Success to true.
	Success bool `json:"success"`
	// Warnings contains non-fatal issues encountered during the operation.
	Warnings []*Diagnostic `json:"warnings,omitempty"`
	// Details holds operation-specific result data.
	Details map[string]interface{} `json:"details,omitempty"`
}

// HasWarnings reports whether the result contains any warnings.
func (r *Result) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// AddWarning appends a warning to the result.
func (r *Result) AddWarning(w *Diagnostic) {
	r.Warnings = append(r.Warnings, w)
}

// AddDetail sets a key-value detail in the result.
func (r *Result) AddDetail(key string, value interface{}) {
	if r.Details == nil {
		r.Details = make(map[string]interface{})
	}
	r.Details[key] = value
}

// Collection holds multiple diagnostics, typically accumulated during
// a multi-step operation.
type Collection struct {
	Diagnostics []*Diagnostic
}

// Append adds a diagnostic to the collection.
func (c *Collection) Append(d *Diagnostic) {
	c.Diagnostics = append(c.Diagnostics, d)
}

// HasErrors reports whether the collection contains any error-severity diagnostics.
func (c *Collection) HasErrors() bool {
	for _, d := range c.Diagnostics {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Errors returns all error-severity diagnostics.
func (c *Collection) Errors() []*Diagnostic {
	var errs []*Diagnostic
	for _, d := range c.Diagnostics {
		if d.Severity == SeverityError {
			errs = append(errs, d)
		}
	}
	return errs
}

// Warnings returns all warning-severity diagnostics.
func (c *Collection) Warnings() []*Diagnostic {
	var warns []*Diagnostic
	for _, d := range c.Diagnostics {
		if d.Severity == SeverityWarning {
			warns = append(warns, d)
		}
	}
	return warns
}

// IsPartialSuccess reports whether there are warnings but no errors.
func (c *Collection) IsPartialSuccess() bool {
	return !c.HasErrors() && len(c.Warnings()) > 0
}
