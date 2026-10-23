package main

import (
	"bytes"
	"debug/macho"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIArguments(t *testing.T) {
	for _, args := range [][]string{
		nil, {"input.txt"}, {"--unknown", "input.c"},
		{"--lex", "--parse", "input.c"}, {"a.c", "b.c"},
	} {
		if err := run(args, io.Discard, io.Discard); err == nil {
			t.Errorf("accepted invalid arguments: %v", args)
		}
	}
	var out bytes.Buffer
	if err := run([]string{"--help"}, &out, io.Discard); err != nil || !strings.Contains(out.String(), "usage:") {
		t.Fatalf("help: output %q, error %v", out.String(), err)
	}
}

func requireToolchain(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("integration tests require macOS or Linux")
	}
	if _, err := exec.LookPath("gcc"); err != nil {
		t.Skip("integration tests require gcc (or Apple's gcc alias)")
	}
}

func writeSource(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input with spaces.c")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertOnlySource(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("unexpected output files: %v", entries)
	}
}

func TestDriverStagesAndPreprocessing(t *testing.T) {
	requireToolchain(t)
	for _, stage := range []string{"--lex", "--parse", "--codegen"} {
		t.Run(stage, func(t *testing.T) {
			path := writeSource(t, "#define ANSWER 42\nint main(void) { /* comment */ return ANSWER; }\n")
			var stderr bytes.Buffer
			// Flags can appear before or after the source path.
			if err := run([]string{path, stage}, io.Discard, &stderr); err != nil {
				t.Fatalf("%v: %s", err, &stderr)
			}
			assertOnlySource(t, path)
		})
	}
	path := writeSource(t, "int main(void) { return ; }")
	if err := run([]string{"--lex", path}, io.Discard, io.Discard); err != nil {
		t.Fatal("lex should accept syntactically invalid token streams:", err)
	}
	if err := run([]string{"--parse", path}, io.Discard, io.Discard); err == nil {
		t.Fatal("parse accepted a missing expression")
	}
	assertOnlySource(t, path)
}

func TestDriverFailuresLeaveNoOutput(t *testing.T) {
	requireToolchain(t)
	for _, source := range []string{
		"#error deliberate preprocessing failure\n",
		"int main(void) { return 1foo; }",
		"int main(void) { return 0 }",
		"int other(void) { return 0; }", // Valid syntax, but cannot link without main.
	} {
		t.Run(source, func(t *testing.T) {
			path := writeSource(t, source)
			if err := run([]string{path}, io.Discard, io.Discard); err == nil {
				t.Fatal("expected compilation to fail")
			}
			assertOnlySource(t, path)
		})
	}
}

func TestDriverAssemblyOutput(t *testing.T) {
	requireToolchain(t)
	path := writeSource(t, "int main(void) { return 42; }")
	if err := run([]string{"-S", path}, io.Discard, io.Discard); err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSuffix(path, ".c")
	assembly, err := os.ReadFile(base + ".s")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(assembly), "movl $42, %eax") {
		t.Fatalf("unexpected assembly: %s", assembly)
	}
	if _, err := os.Stat(base); !os.IsNotExist(err) {
		t.Fatalf("-S must not create an executable: %v", err)
	}
}

func TestDriverBuildAndRun(t *testing.T) {
	requireToolchain(t)
	path := writeSource(t, "int main(void) { return 42; }")
	var stderr bytes.Buffer
	if err := run([]string{path}, io.Discard, &stderr); err != nil {
		t.Fatalf("%v: %s", err, &stderr)
	}
	executable := strings.TrimSuffix(path, ".c")
	if runtime.GOOS == "darwin" {
		binary, err := macho.Open(executable)
		if err != nil {
			t.Fatal(err)
		}
		defer binary.Close()
		if binary.Cpu != macho.CpuAmd64 {
			t.Fatalf("generated CPU %v, want x86-64", binary.Cpu)
		}
	}
	if _, err := os.Stat(executable + ".s"); !os.IsNotExist(err) {
		t.Fatalf("unexpected assembly file: %v", err)
	}
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		if err := exec.Command("/usr/bin/arch", "-x86_64", "/usr/bin/true").Run(); err != nil {
			t.Skip("x86-64 executable built and inspected; running it requires Rosetta")
		}
	}
	err := exec.Command(executable).Run()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 42 {
		t.Fatalf("expected exit status 42, got %v", err)
	}
}
