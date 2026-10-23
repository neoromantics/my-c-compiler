package compiler

import (
	"fmt"
	"strconv"
)

type parser struct {
	tokens []Token
	pos    int
}

// Parse recognizes: int <identifier> ( void ) { return <integer> ; }
func Parse(tokens []Token) (Program, error) {
	p := parser{tokens: tokens}
	function, err := p.parseFunction()
	if err != nil {
		return Program{}, err
	}
	if _, err := p.expect(EOF); err != nil {
		return Program{}, err
	}
	if p.pos != len(tokens) {
		return Program{}, fmt.Errorf("parse: unexpected tokens after end of input")
	}
	return Program{Function: function}, nil
}

func (p *parser) parseFunction() (Function, error) {
	if _, err := p.expect(Int); err != nil {
		return Function{}, err
	}
	name, err := p.expect(Identifier)
	if err != nil {
		return Function{}, err
	}
	for _, kind := range []TokenKind{OpenParen, Void, CloseParen, OpenBrace} {
		if _, err := p.expect(kind); err != nil {
			return Function{}, err
		}
	}
	body, err := p.parseStatement()
	if err != nil {
		return Function{}, err
	}
	if _, err := p.expect(CloseBrace); err != nil {
		return Function{}, err
	}
	return Function{Name: name.Text, Body: body}, nil
}

func (p *parser) parseStatement() (ReturnStatement, error) {
	if _, err := p.expect(Return); err != nil {
		return ReturnStatement{}, err
	}
	value, err := p.parseExpression()
	if err != nil {
		return ReturnStatement{}, err
	}
	if _, err := p.expect(Semicolon); err != nil {
		return ReturnStatement{}, err
	}
	return ReturnStatement{Value: value}, nil
}

func (p *parser) parseExpression() (Constant, error) {
	token, err := p.expect(Integer)
	if err != nil {
		return Constant{}, err
	}
	value, err := strconv.ParseInt(token.Text, 10, 32)
	if err != nil {
		return Constant{}, fmt.Errorf("parse:%d:%d: integer %q is outside the supported int range", token.Line, token.Column, token.Text)
	}
	return Constant{Value: int32(value)}, nil
}

func (p *parser) expect(kind TokenKind) (Token, error) {
	if p.pos >= len(p.tokens) {
		return Token{}, fmt.Errorf("parse: expected %s, found end of input", kind)
	}
	token := p.tokens[p.pos]
	if token.Kind != kind {
		return Token{}, fmt.Errorf("parse:%d:%d: expected %s, found %s %q", token.Line, token.Column, kind, token.Kind, token.Text)
	}
	p.pos++
	return token, nil
}
