package operatordiag

import (
	"os"
	"runtime"
	"testing"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

func TestCheckProcessExists(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process existence check is Linux-specific")
	}

	// Test with current process (should exist).
	if d := CheckProcessExists(os.Getpid()); d != nil {
		t.Errorf("expected no diagnostic for current process, got: %v", d)
	}

	// Test with non-existent process.
	if d := CheckProcessExists(999999); d == nil {
		t.Error("expected diagnostic for non-existent process")
	}
}

func TestCheckYamaPtraceScope(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Yama ptrace scope check is Linux-specific")
	}

	// This test may or may not produce a diagnostic depending on the system.
	// Just verify it doesn't panic.
	_ = CheckYamaPtraceScope()
}

func TestCheckLinuxLockdown(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux lockdown check is Linux-specific")
	}

	// Just verify it doesn't panic.
	_ = CheckLinuxLockdown()
}

func TestCheckStrictDevMem(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("strict devmem check is Linux-specific")
	}

	// Just verify it doesn't panic.
	_ = CheckStrictDevMem()
}

func TestRunPhysicalPreflight(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("physical preflight checks are Linux-specific")
	}

	col := RunPhysicalPreflight()
	if col == nil {
		t.Fatal("expected non-nil collection")
	}

	// On non-Linux or unprivileged systems, we may get diagnostics.
	// Just verify the function runs without panic.
	t.Logf("Preflight diagnostics: %d", len(col.Diagnostics))
}

func TestRunProcessPreflight(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("process preflight checks are Linux-specific")
	}

	col := RunProcessPreflight(os.Getpid())
	if col == nil {
		t.Fatal("expected non-nil collection")
	}

	// Current process should be accessible, but Yama may restrict.
	t.Logf("Preflight diagnostics: %d", len(col.Diagnostics))
}

func TestCheckNamespaces(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("namespace check is Linux-specific")
	}

	// Current process should be in the same namespace as itself.
	if d := CheckNamespaces(os.Getpid()); d != nil {
		t.Errorf("expected no diagnostic for same namespace, got: %v", d)
	}
}

func TestCheckProcessDumpability(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("dumpability check is Linux-specific")
	}

	// Current process should be dumpable.
	if d := CheckProcessDumpability(os.Getpid()); d != nil {
		t.Logf("dumpability diagnostic (may be expected in restricted environments): %v", d)
	}
}

func TestDiagnosticCollection(t *testing.T) {
	col := &diagnostic.Collection{}

	if col.HasErrors() {
		t.Error("expected no errors in empty collection")
	}

	col.Append(diagnostic.Warning(diagnostic.CategorySource, "test warning"))
	if col.HasErrors() {
		t.Error("expected no errors after adding warning")
	}
	if !col.IsPartialSuccess() {
		t.Error("expected partial success with warnings and no errors")
	}

	col.Append(diagnostic.SourceError("test error"))
	if !col.HasErrors() {
		t.Error("expected errors after adding error")
	}
	if col.IsPartialSuccess() {
		t.Error("expected not partial success when errors exist")
	}

	if len(col.Errors()) != 1 {
		t.Errorf("expected 1 error, got %d", len(col.Errors()))
	}
	if len(col.Warnings()) != 1 {
		t.Errorf("expected 1 warning, got %d", len(col.Warnings()))
	}
}
