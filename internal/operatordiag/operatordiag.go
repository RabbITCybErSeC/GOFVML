// Package operatordiag provides operator-facing diagnostics for common
// Linux memory acquisition issues.
package operatordiag

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/RabbITCybErSeC/gofvml/internal/diagnostic"
)

// CheckLinuxLockdown checks if the kernel is in lockdown mode.
// Returns a diagnostic if lockdown is active.
func CheckLinuxLockdown() *diagnostic.Diagnostic {
	data, err := os.ReadFile("/sys/kernel/security/lockdown")
	if err != nil {
		// Lockdown file may not exist on older kernels.
		return nil
	}

	content := strings.TrimSpace(string(data))
	if strings.Contains(content, "[integrity]") || strings.Contains(content, "[confidentiality]") {
		return diagnostic.PolicyError("kernel lockdown is active").
			WithOperation("physical acquisition").
			WithTarget("/sys/kernel/security/lockdown").
			WithSuggestion("lockdown mode restricts memory access; disable lockdown or use alternative acquisition methods")
	}

	return nil
}

// CheckStrictDevMem checks if /dev/mem access is restricted.
// Returns a diagnostic if strict devmem is enabled.
func CheckStrictDevMem() *diagnostic.Diagnostic {
	data, err := os.ReadFile("/proc/sys/kernel/strict_devmem")
	if err != nil {
		return nil
	}

	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil
	}

	if val != 0 {
		return diagnostic.PolicyError("strict devmem is enabled").
			WithOperation("physical acquisition").
			WithTarget("/proc/sys/kernel/strict_devmem").
			WithSuggestion("strict devmem restricts /dev/mem access; use /dev/crash or /proc/kcore instead, or disable strict_devmem")
	}

	return nil
}

// CheckDevCrash checks if /dev/crash is available.
// Returns a diagnostic if it is missing.
func CheckDevCrash() *diagnostic.Diagnostic {
	if _, err := os.Stat("/dev/crash"); err != nil {
		return diagnostic.SourceError("/dev/crash not available").
			WithOperation("physical acquisition").
			WithTarget("/dev/crash").
			WithSuggestion("/dev/crash requires kernel crash dump support; use /proc/kcore or /dev/mem as fallback")
	}
	return nil
}

// CheckProcKcore checks if /proc/kcore is accessible.
// Returns a diagnostic if it is not readable.
func CheckProcKcore() *diagnostic.Diagnostic {
	fi, err := os.Stat("/proc/kcore")
	if err != nil {
		return diagnostic.SourceError("/proc/kcore not accessible").
			WithOperation("physical acquisition").
			WithTarget("/proc/kcore").
			WithCause(err).
			WithSuggestion("/proc/kcore requires root privileges; ensure you are running as root")
	}

	// Check if readable.
	file, err := os.Open("/proc/kcore")
	if err != nil {
		return diagnostic.SourceError("cannot open /proc/kcore").
			WithOperation("physical acquisition").
			WithTarget("/proc/kcore").
			WithCause(err).
			WithSuggestion("/proc/kcore requires root privileges; ensure you are running as root")
	}
	file.Close()

	// /proc/kcore should be a large file (size may be -1 on some systems).
	if fi.Size() == 0 {
		return diagnostic.SourceError("/proc/kcore is empty").
			WithOperation("physical acquisition").
			WithTarget("/proc/kcore").
			WithSuggestion("empty /proc/kcore may indicate kernel configuration issues")
	}

	return nil
}

// CheckYamaPtraceScope checks the Yama ptrace scope.
// Returns a diagnostic if ptrace is restricted.
func CheckYamaPtraceScope() *diagnostic.Diagnostic {
	data, err := os.ReadFile("/proc/sys/kernel/yama/ptrace_scope")
	if err != nil {
		// Yama may not be available.
		return nil
	}

	val, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil
	}

	switch val {
	case 1:
		return diagnostic.PolicyError("ptrace is restricted (yama.ptrace_scope=1)").
			WithOperation("process acquisition").
			WithTarget("/proc/sys/kernel/yama/ptrace_scope").
			WithSuggestion("process memory access requires CAP_SYS_PTRACE or the target process to be a descendant; run with appropriate privileges")
	case 2:
		return diagnostic.PolicyError("ptrace is disabled (yama.ptrace_scope=2)").
			WithOperation("process acquisition").
			WithTarget("/proc/sys/kernel/yama/ptrace_scope").
			WithSuggestion("process memory access requires CAP_SYS_PTRACE; run as root or adjust yama.ptrace_scope")
	case 3:
		return diagnostic.PolicyError("ptrace is completely disabled (yama.ptrace_scope=3)").
			WithOperation("process acquisition").
			WithTarget("/proc/sys/kernel/yama/ptrace_scope").
			WithSuggestion("process memory access is not possible with current settings; adjust kernel boot parameters")
	}

	return nil
}

// CheckProcessDumpability checks if a process is dumpable.
// Returns a diagnostic if the process is not dumpable.
func CheckProcessDumpability(pid int) *diagnostic.Diagnostic {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return diagnostic.ProcessError("cannot read process status").
			WithOperation("process acquisition").
			WithTarget(fmt.Sprintf("/proc/%d", pid)).
			WithCause(err).
			WithSuggestion("the process may have exited or you lack permission to access it")
	}

	content := string(data)
	if strings.Contains(content, "VmFlags:") {
		// Check for dumpable flag.
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "VmFlags:") {
				if !strings.Contains(line, "dd") {
					return diagnostic.PolicyError("process is not dumpable").
						WithOperation("process acquisition").
						WithTarget(fmt.Sprintf("pid=%d", pid)).
						WithSuggestion(fmt.Sprintf("the process may have changed its dumpability; check /proc/%d/status for Dumpable field", pid))
				}
			}
		}
	}

	// Check for non-dumpable processes using the traditional approach.
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Uid:") {
			// If we can't read /proc/pid/mem, the process may not be dumpable.
			memPath := fmt.Sprintf("/proc/%d/mem", pid)
			f, err := os.Open(memPath)
			if err != nil {
				return diagnostic.PolicyError("cannot access process memory").
					WithOperation("process acquisition").
					WithTarget(memPath).
					WithCause(err).
					WithSuggestion("ensure the process exists, has not exited, and you have ptrace access")
			}
			f.Close()
			break
		}
	}

	return nil
}

// CheckProcessExists checks if a process still exists.
// Returns a diagnostic if the process is gone.
func CheckProcessExists(pid int) *diagnostic.Diagnostic {
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err != nil {
		return diagnostic.ProcessError("process does not exist or has exited").
			WithOperation("process acquisition").
			WithTarget(fmt.Sprintf("pid=%d", pid)).
			WithCause(err)
	}
	return nil
}

// CheckNamespaces checks if the target process is in a different namespace.
// Returns a diagnostic if namespace separation may prevent access.
func CheckNamespaces(targetPID int) *diagnostic.Diagnostic {
	selfPID := os.Getpid()

	// Compare PID namespaces.
	selfNS, err1 := os.Readlink(fmt.Sprintf("/proc/%d/ns/pid", selfPID))
	targetNS, err2 := os.Readlink(fmt.Sprintf("/proc/%d/ns/pid", targetPID))

	if err1 != nil || err2 != nil {
		return nil // Can't determine, don't warn.
	}

	if selfNS != targetNS {
		return diagnostic.PolicyError("target process is in a different PID namespace").
			WithOperation("process acquisition").
			WithTarget(fmt.Sprintf("pid=%d", targetPID)).
			WithSuggestion("cross-namespace process memory access requires special privileges; enter the target namespace or use nsenter")
	}

	return nil
}

// RunPhysicalPreflight runs a preflight check for physical acquisition.
// It returns a collection of diagnostics for common issues.
func RunPhysicalPreflight() *diagnostic.Collection {
	col := &diagnostic.Collection{}

	if d := CheckLinuxLockdown(); d != nil {
		col.Append(d)
	}
	if d := CheckStrictDevMem(); d != nil {
		col.Append(d)
	}
	if d := CheckDevCrash(); d != nil {
		col.Append(d)
	}
	if d := CheckProcKcore(); d != nil {
		col.Append(d)
	}

	return col
}

// RunProcessPreflight runs a preflight check for process acquisition.
// It returns a collection of diagnostics for common issues.
func RunProcessPreflight(pid int) *diagnostic.Collection {
	col := &diagnostic.Collection{}

	if d := CheckYamaPtraceScope(); d != nil {
		col.Append(d)
	}
	if d := CheckProcessExists(pid); d != nil {
		col.Append(d)
	}
	if d := CheckProcessDumpability(pid); d != nil {
		col.Append(d)
	}
	if d := CheckNamespaces(pid); d != nil {
		col.Append(d)
	}

	return col
}
