package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"my-c-compiler/internal/compiler"
)

const usage = "usage: mycc [--lex | --parse | --codegen | -S] <file.c>"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mycc:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	var sourcePath, stage string
	for _, arg := range args {
		switch arg {
		case "-h", "--help":
			fmt.Fprintln(stdout, usage)
			return nil
		case "--lex", "--parse", "--codegen", "-S":
			if stage != "" {
				return fmt.Errorf("choose only one stage flag\n%s", usage)
			}
			stage = arg
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown option %q\n%s", arg, usage)
			}
			if sourcePath != "" {
				return fmt.Errorf("expected exactly one input file\n%s", usage)
			}
			sourcePath = arg
		}
	}
	if sourcePath == "" || filepath.Ext(sourcePath) != ".c" {
		return fmt.Errorf("expected a .c input file\n%s", usage)
	}
	// Absolute paths also keep gcc from interpreting filenames as command options.
	sourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return err
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported operating system %s", runtime.GOOS)
	}

	// Keep preprocessed source in memory and temporary assembly in a private
	// directory. Stage checks and failed compiles leave no intermediate files.
	preprocess := exec.Command("gcc", targetArgs("-E", "-P", sourcePath)...)
	preprocess.Stderr = stderr
	source, err := preprocess.Output()
	if err != nil {
		return fmt.Errorf("preprocessing %s: %w", sourcePath, err)
	}
	tokens, err := compiler.Lex(string(source))
	if err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	if stage == "--lex" {
		return nil
	}
	program, err := compiler.Parse(tokens)
	if err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	if stage == "--parse" {
		return nil
	}
	assembly := compiler.Generate(program)
	if stage == "--codegen" {
		return nil
	}
	text, err := compiler.Emit(assembly, runtime.GOOS)
	if err != nil {
		return err
	}
	outputPath := strings.TrimSuffix(sourcePath, ".c")
	if stage == "-S" {
		return os.WriteFile(outputPath+".s", []byte(text), 0644)
	}
	tempDir, err := os.MkdirTemp(filepath.Dir(sourcePath), ".mycc-*")
	if err != nil {
		return fmt.Errorf("creating build directory: %w", err)
	}
	defer os.RemoveAll(tempDir)
	assemblyPath := filepath.Join(tempDir, "program.s")
	if err := os.WriteFile(assemblyPath, []byte(text), 0600); err != nil {
		return err
	}
	executablePath := filepath.Join(tempDir, "program")
	link := exec.Command("gcc", targetArgs(assemblyPath, "-o", executablePath)...)
	link.Stdout, link.Stderr = stdout, stderr
	if err := link.Run(); err != nil {
		return fmt.Errorf("assembling and linking: %w", err)
	}
	// Publish only a successfully linked executable, on the same filesystem.
	return os.Rename(executablePath, outputPath)
}

func targetArgs(args ...string) []string {
	if runtime.GOOS == "darwin" {
		return append([]string{"-arch", "x86_64"}, args...)
	}
	return append([]string{"-m64"}, args...)
}
