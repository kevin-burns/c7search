package cli

import (
	"bytes"
	"strings"
	"testing"
)

// T8: `c7search --version` and `c7search version` must produce the same
// build identifier on stdout. (Cobra's Version field powers --version
// without an explicit flag declaration; this test guards the wiring.)
func TestRoot_VersionFlag(t *testing.T) {
	var stdoutFlag, stdoutCmd bytes.Buffer
	if exit := run(&stdoutFlag, &bytes.Buffer{}, []string{"--version"}); exit != 0 {
		t.Fatalf("--version exit %d", exit)
	}
	if exit := run(&stdoutCmd, &bytes.Buffer{}, []string{"version"}); exit != 0 {
		t.Fatalf("version exit %d", exit)
	}
	if strings.TrimSpace(stdoutFlag.String()) != strings.TrimSpace(stdoutCmd.String()) {
		t.Errorf("--version and version disagree:\n  flag: %q\n  cmd:  %q",
			stdoutFlag.String(), stdoutCmd.String())
	}
	if !strings.Contains(stdoutFlag.String(), "c7search") {
		t.Errorf("--version output missing program name: %q", stdoutFlag.String())
	}
}

// T1: flags must not bleed across two Execute()-style invocations of the
// command tree. The first run sets --api-key=A; the second run, with no
// --api-key, must observe an empty key (it should fall back to env, which
// the test also clears). t.Setenv guards parallelism — Go enforces that
// t.Setenv-using tests cannot run with t.Parallel().
func TestRootCmd_NoFlagBleed(t *testing.T) {
	t.Setenv("CONTEXT7_API_KEY", "")

	root1 := newRootCmd()
	root1.SetOut(&bytes.Buffer{})
	root1.SetErr(&bytes.Buffer{})
	root1.SetArgs([]string{"--api-key", "ctx7sk-FIRST", "version"})
	if err := root1.Execute(); err != nil {
		t.Fatalf("first execute: %v", err)
	}

	root2 := newRootCmd()
	root2.SetOut(&bytes.Buffer{})
	root2.SetErr(&bytes.Buffer{})
	root2.SetArgs([]string{"version"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("second execute: %v", err)
	}

	got := resolveAPIKey(root2)
	if got != "" {
		t.Fatalf("flag bleed: second invocation saw key %q, want empty", got)
	}
}
