package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

func TestRenderDiagnostics(t *testing.T) {
	var buf bytes.Buffer

	diagnostics := []*diagnostic.Diagnostic{
		diagnostic.New(diagnostic.SeverityError, diagnostic.CategorySource, "source failed").
			WithOperation("physical acquisition").
			WithTarget("/dev/mem").
			WithSuggestion("run as root"),
		diagnostic.New(diagnostic.SeverityWarning, diagnostic.CategoryPolicy, "partial read").
			WithOperation("physical acquisition").
			WithTarget("0x1000-0x2000"),
		diagnostic.New(diagnostic.SeverityInfo, diagnostic.CategorySource, "using /dev/crash").
			WithOperation("physical acquisition"),
	}

	RenderDiagnostics(&buf, diagnostics)
	output := buf.String()

	if !strings.Contains(output, "Errors:") {
		t.Error("expected 'Errors:' section")
	}
	if !strings.Contains(output, "Warnings:") {
		t.Error("expected 'Warnings:' section")
	}
	if !strings.Contains(output, "Info:") {
		t.Error("expected 'Info:' section")
	}
	if !strings.Contains(output, "source failed") {
		t.Error("expected error message")
	}
	if !strings.Contains(output, "run as root") {
		t.Error("expected suggestion")
	}
}

func TestRenderDiagnosticsEmpty(t *testing.T) {
	var buf bytes.Buffer
	RenderDiagnostics(&buf, nil)
	if buf.Len() != 0 {
		t.Error("expected empty output for nil diagnostics")
	}
}

func TestRenderResult(t *testing.T) {
	tests := []struct {
		name       string
		result     *diagnostic.Result
		wantOutput []string
	}{
		{
			name:       "success",
			result:     &diagnostic.Result{Success: true},
			wantOutput: []string{"Result: success"},
		},
		{
			name: "partial_success",
			result: &diagnostic.Result{
				Success: true,
				Warnings: []*diagnostic.Diagnostic{
					diagnostic.Warning(diagnostic.CategorySource, "partial"),
				},
			},
			wantOutput: []string{"Result: partial success", "partial"},
		},
		{
			name:       "failure",
			result:     &diagnostic.Result{Success: false},
			wantOutput: []string{"Result: failed"},
		},
		{
			name: "with_details",
			result: &diagnostic.Result{
				Success: true,
				Details: map[string]interface{}{
					"bytes_written": uint64(1024),
				},
			},
			wantOutput: []string{"Result: success", "bytes_written", "1024"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderResult(&buf, tt.result)
			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestRenderCollection(t *testing.T) {
	tests := []struct {
		name       string
		collection *diagnostic.Collection
		wantOutput []string
	}{
		{
			name:       "nil",
			collection: nil,
			wantOutput: []string{"No diagnostics."},
		},
		{
			name:       "empty",
			collection: &diagnostic.Collection{},
			wantOutput: []string{"No diagnostics."},
		},
		{
			name:       "success_with_nil_diagnostics",
			collection: &diagnostic.Collection{},
			wantOutput: []string{"No diagnostics."},
		},
		{
			name: "with_warnings",
			collection: &diagnostic.Collection{
				Diagnostics: []*diagnostic.Diagnostic{
					diagnostic.Warning(diagnostic.CategorySource, "warning1"),
				},
			},
			wantOutput: []string{"Operation completed with 1 warning(s).", "warning1"},
		},
		{
			name: "with_errors",
			collection: &diagnostic.Collection{
				Diagnostics: []*diagnostic.Diagnostic{
					diagnostic.SourceError("error1"),
					diagnostic.Warning(diagnostic.CategorySource, "warning1"),
				},
			},
			wantOutput: []string{"Operation completed with 1 error(s) and 1 warning(s).", "error1", "warning1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			RenderCollection(&buf, tt.collection)
			output := buf.String()
			for _, want := range tt.wantOutput {
				if !strings.Contains(output, want) {
					t.Errorf("expected output to contain %q, got:\n%s", want, output)
				}
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		name string
		c    *diagnostic.Collection
		want int
	}{
		{"nil", nil, 0},
		{"empty", &diagnostic.Collection{}, 0},
		{"success", &diagnostic.Collection{Diagnostics: []*diagnostic.Diagnostic{}}, 0},
		{"warnings", &diagnostic.Collection{Diagnostics: []*diagnostic.Diagnostic{diagnostic.Warning(diagnostic.CategorySource, "w")}}, 0},
		{"errors", &diagnostic.Collection{Diagnostics: []*diagnostic.Diagnostic{diagnostic.SourceError("e")}}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.c); got != tt.want {
				t.Errorf("ExitCode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRenderDiagnosticWithCause(t *testing.T) {
	var buf bytes.Buffer
	d := diagnostic.New(diagnostic.SeverityError, diagnostic.CategorySource, "open failed").
		WithCause(errors.New("permission denied"))
	RenderDiagnostics(&buf, []*diagnostic.Diagnostic{d})
	output := buf.String()
	if !strings.Contains(output, "cause: permission denied") {
		t.Errorf("expected cause in output, got:\n%s", output)
	}
}
