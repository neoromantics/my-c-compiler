// Package compiler implements the compiler passes from Writing a C Compiler.
package compiler

import "fmt"

type TokenKind string

const (
	Identifier TokenKind = "identifier"
	Integer    TokenKind = "integer"
	Int        TokenKind = "int"
	Void       TokenKind = "void"
	Return     TokenKind = "return"
	OpenParen  TokenKind = "("
	CloseParen TokenKind = ")"
	OpenBrace  TokenKind = "{"
	CloseBrace TokenKind = "}"
	Semicolon  TokenKind = ";"
	EOF        TokenKind = "end of input"
)

type Token struct {
	Kind   TokenKind
	Text   string
	Line   int
	Column int
}

// Lex tokenizes preprocessed C. Comments and directives are handled by gcc -E.
// Locations refer to the preprocessed input.
func Lex(source string) ([]Token, error) {
	var tokens []Token
	line, column := 1, 1
	for i := 0; i < len(source); {
		c := source[i]
		if isSpace(c) {
			if c == '\n' {
				line, column = line+1, 1
			} else {
				column++
			}
			i++
			continue
		}

		start := i
		var kind TokenKind
		switch {
		case isIdentifierStart(c):
			i++
			for i < len(source) && isIdentifierContinue(source[i]) {
				i++
			}
			kind = Identifier
			switch source[start:i] {
			case "int":
				kind = Int
			case "void":
				kind = Void
			case "return":
				kind = Return
			}
		case isDigit(c):
			i++
			for i < len(source) && isDigit(source[i]) {
				i++
			}
			// Do not split an invalid token such as 1foo into 1 and foo.
			if i < len(source) && isIdentifierStart(source[i]) {
				return nil, fmt.Errorf("lex:%d:%d: invalid integer token", line, column)
			}
			kind = Integer
		default:
			switch c {
			case '(', ')', '{', '}', ';':
				kind = TokenKind(string(c))
				i++
			default:
				return nil, fmt.Errorf("lex:%d:%d: unexpected character %q", line, column, c)
			}
		}
		tokens = append(tokens, Token{Kind: kind, Text: source[start:i], Line: line, Column: column})
		column += i - start
	}
	return append(tokens, Token{Kind: EOF, Line: line, Column: column}), nil
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isIdentifierStart(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_'
}

func isIdentifierContinue(c byte) bool { return isIdentifierStart(c) || isDigit(c) }
