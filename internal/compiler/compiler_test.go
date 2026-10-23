package compiler

import (
	"fmt"
	"strings"
	"testing"
)

func TestLexBoundaries(t *testing.T) {
	tokens, err := Lex("int integer void voided return return_value _main2 123\n{}();")
	if err != nil {
		t.Fatal(err)
	}
	want := []TokenKind{Int, Identifier, Void, Identifier, Return, Identifier, Identifier, Integer, OpenBrace, CloseBrace, OpenParen, CloseParen, Semicolon, EOF}
	if len(tokens) != len(want) {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want))
	}
	for i, kind := range want {
		if tokens[i].Kind != kind {
			t.Errorf("token %d: got %s, want %s", i, tokens[i].Kind, kind)
		}
	}
	if tokens[8].Line != 2 || tokens[8].Column != 1 {
		t.Errorf("wrong location after newline: %+v", tokens[8])
	}
}

func TestLexRejectsInvalidTokens(t *testing.T) {
	for _, source := range []string{"1foo", "2_bar", "@", "`", "\\", "é", "0.5"} {
		t.Run(source, func(t *testing.T) {
			if _, err := Lex(source); err == nil {
				t.Fatal("accepted invalid token")
			}
		})
	}
}

func TestParseConstantsAndNames(t *testing.T) {
	for _, value := range []int32{0, 42, 255, 256, 2147483647} {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			tokens, err := Lex(fmt.Sprintf("int _answer2(void) { return %d; }", value))
			if err != nil {
				t.Fatal(err)
			}
			program, err := Parse(tokens)
			if err != nil {
				t.Fatal(err)
			}
			if program.Function.Name != "_answer2" || program.Function.Body.Value.Value != value {
				t.Fatalf("unexpected AST: %+v", program)
			}
		})
	}
}

func TestParseRejectsInvalidPrograms(t *testing.T) {
	for _, source := range []string{
		"", "int", "int main(void) { return", "int main(void) { return 0;",
		"int main() { return 0; }", "int main(void) { return 0 }",
		"int 3(void) { return 0; }", "int main(void) { return int; }",
		"int main(void) { return 0; } junk",
		"int main(void) { return 0; } int f(void) { return 1; }",
		"int main(void) { return 0; return 1; }",
		"int main(void) { return 2147483648; }",
		"int main(void) { return 9999999999999999999999999999999; }",
	} {
		t.Run(source, func(t *testing.T) {
			tokens, err := Lex(source)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Parse(tokens); err == nil {
				t.Fatal("accepted invalid program")
			}
		})
	}
}

func TestParseTruncatedTokenStream(t *testing.T) {
	tokens, err := Lex("int main(void) { return 0; }")
	if err != nil {
		t.Fatal(err)
	}
	for end := 0; end < len(tokens); end++ {
		if _, err := Parse(tokens[:end]); err == nil {
			t.Errorf("accepted token stream truncated at %d", end)
		}
	}
}

func TestEmitTargets(t *testing.T) {
	tokens, err := Lex("int main(void) { return 42; }")
	if err != nil {
		t.Fatal(err)
	}
	program, err := Parse(tokens)
	if err != nil {
		t.Fatal(err)
	}
	assembly := Generate(program)
	for _, target := range []struct {
		os, symbol, suffix string
	}{
		{"darwin", "_main", ""},
		{"linux", "main", "\t.section .note.GNU-stack,\"\",@progbits\n"},
	} {
		t.Run(target.os, func(t *testing.T) {
			got, err := Emit(assembly, target.os)
			if err != nil {
				t.Fatal(err)
			}
			want := fmt.Sprintf("\t.text\n\t.globl %s\n%s:\n\tmovl $42, %%eax\n\tret\n%s", target.symbol, target.symbol, target.suffix)
			if got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
	if _, err := Emit(assembly, "unsupported"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported-target error, got %v", err)
	}
}
