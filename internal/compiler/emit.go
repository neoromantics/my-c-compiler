package compiler

import (
	"fmt"
	"strings"
)

// Emit renders x86-64 assembly in AT&T syntax for Linux or macOS.
func Emit(program AssemblyProgram, targetOS string) (string, error) {
	if targetOS != "linux" && targetOS != "darwin" {
		return "", fmt.Errorf("unsupported target OS %q (expected linux or darwin)", targetOS)
	}
	name := program.Function.Name
	if targetOS == "darwin" {
		name = "_" + name
	}
	var out strings.Builder
	fmt.Fprintf(&out, "\t.text\n\t.globl %s\n%s:\n", name, name)
	for _, instruction := range program.Function.Instructions {
		switch instruction := instruction.(type) {
		case Mov:
			fmt.Fprintf(&out, "\tmovl $%d, %%%s\n", instruction.Source.Value, instruction.Destination)
		case Ret:
			out.WriteString("\tret\n")
		default:
			return "", fmt.Errorf("unsupported assembly instruction %T", instruction)
		}
	}
	if targetOS == "linux" {
		out.WriteString("\t.section .note.GNU-stack,\"\",@progbits\n")
	}
	return out.String(), nil
}
