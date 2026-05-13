package diagnostic

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestDiagnosticError(t *testing.T) {
	d := New(SeverityError, CategorySource, "unable to open source").
		WithOperation("physical acquisition").
		WithTarget("/dev/mem").
		WithSuggestion("run with root privileges").
		WithCause(errors.New("permission denied"))

	got := d.Error()
	want := "[error:source] physical acquisition: unable to open source (target: /dev/mem); suggestion: run with root privileges: permission denied"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestDiagnosticMinimal(t *testing.T) {
	d := New(SeverityWarning, CategoryParse, "malformed line")
	got := d.Error()
	want := "[warning:parse] malformed line"
	if got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestDiagnosticIs(t *testing.T) {
	d := New(SeverityError, CategorySource, "test")
	if !errors.Is(d, New(SeverityError, CategorySource, "")) {
		t.Error("expected diagnostic to match same severity/category")
	}
	if errors.Is(d, New(SeverityWarning, CategorySource, "")) {
		t.Error("expected diagnostic to not match different severity")
	}
	if errors.Is(d, New(SeverityError, CategoryPolicy, "")) {
		t.Error("expected diagnostic to not match different category")
	}
}

func TestDiagnosticUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	d := New(SeverityError, CategorySource, "test").WithCause(cause)
	if !errors.Is(d, cause) {
		t.Error("expected diagnostic to wrap underlying error")
	}
}

func TestFactoryFunctions(t *testing.T) {
	tests := []struct {
		name     string
		d        *Diagnostic
		wantSev  Severity
		wantCat  Category
		wantMsg  string
	}{
		{
			name:    "SourceError",
			d:       SourceError("source failed"),
			wantSev: SeverityError,
			wantCat: CategorySource,
			wantMsg: "source failed",
		},
		{
			name:    "PolicyError",
			d:       PolicyError("policy failed"),
			wantSev: SeverityError,
			wantCat: CategoryPolicy,
			wantMsg: "policy failed",
		},
		{
			name:    "ParseError",
			d:       ParseError("parse failed"),
			wantSev: SeverityError,
			wantCat: CategoryParse,
			wantMsg: "parse failed",
		},
		{
			name:    "FormatError",
			d:       FormatError("format failed"),
			wantSev: SeverityError,
			wantCat: CategoryFormat,
			wantMsg: "format failed",
		},
		{
			name:    "UploadError",
			d:       UploadError("upload failed"),
			wantSev: SeverityError,
			wantCat: CategoryUpload,
			wantMsg: "upload failed",
		},
		{
			name:    "ProcessError",
			d:       ProcessError("process failed"),
			wantSev: SeverityError,
			wantCat: CategoryProcess,
			wantMsg: "process failed",
		},
		{
			name:    "Warning",
			d:       Warning(CategorySource, "source warning"),
			wantSev: SeverityWarning,
			wantCat: CategorySource,
			wantMsg: "source warning",
		},
		{
			name:    "Info",
			d:       Info(CategorySource, "source info"),
			wantSev: SeverityInfo,
			wantCat: CategorySource,
			wantMsg: "source info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.d.Severity != tt.wantSev {
				t.Errorf("Severity = %v, want %v", tt.d.Severity, tt.wantSev)
			}
			if tt.d.Category != tt.wantCat {
				t.Errorf("Category = %v, want %v", tt.d.Category, tt.wantCat)
			}
			if tt.d.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", tt.d.Message, tt.wantMsg)
			}
		})
	}
}

func TestSeverityString(t *testing.T) {
	tests := []struct {
		sev  Severity
		want string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{Severity(99), "severity(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.sev.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCategoryString(t *testing.T) {
	tests := []struct {
		cat  Category
		want string
	}{
		{CategorySource, "source"},
		{CategoryPolicy, "policy"},
		{CategoryParse, "parse"},
		{CategoryFormat, "format"},
		{CategoryUpload, "upload"},
		{CategoryProcess, "process"},
		{CategoryInternal, "internal"},
		{Category(99), "category(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.cat.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResult(t *testing.T) {
	r := Result{Success: true}

	if r.HasWarnings() {
		t.Error("expected no warnings initially")
	}

	r.AddWarning(Warning(CategorySource, "partial read"))
	if !r.HasWarnings() {
		t.Error("expected warnings after adding one")
	}
	if len(r.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(r.Warnings))
	}

	r.AddDetail("bytes_written", uint64(1024))
	if r.Details["bytes_written"] != uint64(1024) {
		t.Errorf("expected bytes_written detail")
	}
}

func TestCollection(t *testing.T) {
	var c Collection

	if c.HasErrors() {
		t.Error("expected no errors in empty collection")
	}
	if c.IsPartialSuccess() {
		t.Error("expected not partial success in empty collection")
	}

	c.Append(Warning(CategorySource, "warning 1"))
	if c.HasErrors() {
		t.Error("expected no errors after adding warning")
	}
	if !c.IsPartialSuccess() {
		t.Error("expected partial success with warnings and no errors")
	}

	c.Append(SourceError("error 1"))
	if !c.HasErrors() {
		t.Error("expected errors after adding error")
	}
	if c.IsPartialSuccess() {
		t.Error("expected not partial success when errors exist")
	}

	if len(c.Errors()) != 1 {
		t.Errorf("expected 1 error, got %d", len(c.Errors()))
	}
	if len(c.Warnings()) != 1 {
		t.Errorf("expected 1 warning, got %d", len(c.Warnings()))
	}
}

func TestDiagnosticJSON(t *testing.T) {
	// Diagnostic should be serializable (indirect test via fmt).
	d := New(SeverityError, CategoryPolicy, "CAP_SYS_ADMIN required").
		WithOperation("physical acquisition").
		WithTarget("/proc/iomem").
		WithSuggestion("run as root or with CAP_SYS_ADMIN")

	// Verify it can be formatted without panicking.
	_ = fmt.Sprintf("%+v", d)
}

func TestDiagnosticMarshalJSON(t *testing.T) {
	d := New(SeverityError, CategoryPolicy, "CAP_SYS_ADMIN required").
		WithOperation("physical acquisition").
		WithTarget("/proc/iomem").
		WithSuggestion("run as root or with CAP_SYS_ADMIN").
		WithCause(errors.New("permission denied"))

	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}

	wantSubstrings := []string{
		`"severity":"error"`,
		`"category":"policy"`,
		`"operation":"physical acquisition"`,
		`"message":"CAP_SYS_ADMIN required"`,
		`"target":"/proc/iomem"`,
		`"suggestion":"run as root or with CAP_SYS_ADMIN"`,
		`"cause":"permission denied"`,
	}

	for _, want := range wantSubstrings {
		if !strings.Contains(string(data), want) {
			t.Errorf("expected JSON to contain %q, got: %s", want, string(data))
		}
	}
}

func TestDiagnosticUnmarshalJSON(t *testing.T) {
	input := `{
		"severity": "warning",
		"category": "source",
		"operation": "physical acquisition",
		"message": "partial read",
		"target": "0x1000-0x2000",
		"suggestion": "check source availability"
	}`

	var d Diagnostic
	if err := json.Unmarshal([]byte(input), &d); err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}

	if d.Severity != SeverityWarning {
		t.Errorf("Severity = %v, want warning", d.Severity)
	}
	if d.Category != CategorySource {
		t.Errorf("Category = %v, want source", d.Category)
	}
	if d.Operation != "physical acquisition" {
		t.Errorf("Operation = %q, want 'physical acquisition'", d.Operation)
	}
	if d.Message != "partial read" {
		t.Errorf("Message = %q, want 'partial read'", d.Message)
	}
	if d.Target != "0x1000-0x2000" {
		t.Errorf("Target = %q, want '0x1000-0x2000'", d.Target)
	}
	if d.Suggestion != "check source availability" {
		t.Errorf("Suggestion = %q, want 'check source availability'", d.Suggestion)
	}
	if d.Cause != nil {
		t.Errorf("expected no cause, got %v", d.Cause)
	}
}

func TestDiagnosticUnmarshalJSONUnknownSeverity(t *testing.T) {
	input := `{"severity": "unknown", "category": "source", "message": "test"}`
	var d Diagnostic
	err := json.Unmarshal([]byte(input), &d)
	if err == nil {
		t.Error("expected error for unknown severity")
	}
}

func TestDiagnosticUnmarshalJSONUnknownCategory(t *testing.T) {
	input := `{"severity": "error", "category": "unknown", "message": "test"}`
	var d Diagnostic
	err := json.Unmarshal([]byte(input), &d)
	if err == nil {
		t.Error("expected error for unknown category")
	}
}

func TestDiagnosticJSONRoundTrip(t *testing.T) {
	original := New(SeverityWarning, CategoryProcess, "partial dump").
		WithOperation("process acquisition").
		WithTarget("1234").
		WithSuggestion("retry with elevated privileges")

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var restored Diagnostic
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if restored.Severity != original.Severity {
		t.Errorf("Severity mismatch: %v vs %v", restored.Severity, original.Severity)
	}
	if restored.Category != original.Category {
		t.Errorf("Category mismatch: %v vs %v", restored.Category, original.Category)
	}
	if restored.Operation != original.Operation {
		t.Errorf("Operation mismatch: %q vs %q", restored.Operation, original.Operation)
	}
	if restored.Message != original.Message {
		t.Errorf("Message mismatch: %q vs %q", restored.Message, original.Message)
	}
	if restored.Target != original.Target {
		t.Errorf("Target mismatch: %q vs %q", restored.Target, original.Target)
	}
	if restored.Suggestion != original.Suggestion {
		t.Errorf("Suggestion mismatch: %q vs %q", restored.Suggestion, original.Suggestion)
	}
}

func TestResultJSON(t *testing.T) {
	r := Result{
		Success: true,
		Warnings: []*Diagnostic{
			Warning(CategorySource, "warning1"),
		},
		Details: map[string]interface{}{
			"bytes_written": uint64(1024),
		},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"success":true`) {
		t.Error("expected success field in JSON")
	}
	if !strings.Contains(string(data), `"warnings"`) {
		t.Error("expected warnings field in JSON")
	}
	if !strings.Contains(string(data), `"details"`) {
		t.Error("expected details field in JSON")
	}
}
