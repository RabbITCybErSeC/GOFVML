// Package cli provides shared command-line parsing and validation for GOFVML tools.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

// RenderDiagnostics writes a human-readable summary of diagnostics to w.
// It groups diagnostics by severity and renders them with optional color hints.
func RenderDiagnostics(w io.Writer, diagnostics []*diagnostic.Diagnostic) {
	if len(diagnostics) == 0 {
		return
	}

	var errors, warnings, infos []*diagnostic.Diagnostic
	for _, d := range diagnostics {
		switch d.Severity {
		case diagnostic.SeverityError:
			errors = append(errors, d)
		case diagnostic.SeverityWarning:
			warnings = append(warnings, d)
		case diagnostic.SeverityInfo:
			infos = append(infos, d)
		}
	}

	if len(errors) > 0 {
		fmt.Fprintf(w, "\nErrors:\n")
		for _, d := range errors {
			renderDiagnostic(w, d, "  ✗ ")
		}
	}

	if len(warnings) > 0 {
		fmt.Fprintf(w, "\nWarnings:\n")
		for _, d := range warnings {
			renderDiagnostic(w, d, "  ⚠ ")
		}
	}

	if len(infos) > 0 {
		fmt.Fprintf(w, "\nInfo:\n")
		for _, d := range infos {
			renderDiagnostic(w, d, "  ℹ ")
		}
	}
}

func renderDiagnostic(w io.Writer, d *diagnostic.Diagnostic, prefix string) {
	fmt.Fprint(w, prefix)

	if d.Operation != "" {
		fmt.Fprintf(w, "[%s] ", d.Operation)
	}

	fmt.Fprintf(w, "%s", d.Message)

	if d.Target != "" {
		fmt.Fprintf(w, " (%s)", d.Target)
	}

	fmt.Fprintln(w)

	if d.Suggestion != "" {
		fmt.Fprintf(w, "%s    → %s\n", strings.Repeat(" ", len(prefix)-2), d.Suggestion)
	}

	if d.Cause != nil {
		fmt.Fprintf(w, "%s    cause: %v\n", strings.Repeat(" ", len(prefix)-2), d.Cause)
	}
}

// RenderResult writes a human-readable summary of a Result to w.
func RenderResult(w io.Writer, r *diagnostic.Result) {
	if r.Success {
		if r.HasWarnings() {
			fmt.Fprintln(w, "Result: partial success")
		} else {
			fmt.Fprintln(w, "Result: success")
		}
	} else {
		fmt.Fprintln(w, "Result: failed")
	}

	if len(r.Details) > 0 {
		fmt.Fprintln(w, "Details:")
		for k, v := range r.Details {
			fmt.Fprintf(w, "  %s: %v\n", k, v)
		}
	}

	RenderDiagnostics(w, r.Warnings)
}

// RenderCollection writes a human-readable summary of a diagnostic collection to w.
func RenderCollection(w io.Writer, c *diagnostic.Collection) {
	if c == nil || len(c.Diagnostics) == 0 {
		fmt.Fprintln(w, "No diagnostics.")
		return
	}

	if c.HasErrors() {
		fmt.Fprintf(w, "Operation completed with %d error(s) and %d warning(s).\n",
			len(c.Errors()), len(c.Warnings()))
	} else if len(c.Warnings()) > 0 {
		fmt.Fprintf(w, "Operation completed with %d warning(s).\n", len(c.Warnings()))
	} else {
		fmt.Fprintln(w, "Operation completed successfully.")
	}

	RenderDiagnostics(w, c.Diagnostics)
}

// ExitCode returns a suggested process exit code based on a diagnostic collection.
func ExitCode(c *diagnostic.Collection) int {
	if c == nil {
		return 0
	}
	if c.HasErrors() {
		return 1
	}
	if len(c.Warnings()) > 0 {
		return 0 // partial success is still success for exit codes
	}
	return 0
}
