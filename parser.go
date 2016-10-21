package ariaconfig

import (
	"fmt"
	"strconv"
)

type parser struct {
	l   *lexer
	buf struct {
		tok item   //last read tok
		lit string //last read literal
		n   int    //buffer size
	}
}

type selectStatement struct {
	key   string
	value string
	typ   itemType //Only for values
}

func newParser(str string) *parser {
	return &parser{l: lex("Lexer", str)}
}

func (p *parser) scan() (tok item, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}
	tok, lit = p.getTokLit()

	return
}

func (p *parser) getTokLit() (tok item, lit string) {
	tok = p.l.nextItem()
	lit = tok.val
	return
}

func (p *parser) unscan() {
	p.buf.n = 1
}

func (p *parser) parse() (*selectStatement, error) {
	stmt := &selectStatement{}

	tok, lit := p.getTokLit()
	if tok.typ == itemLeftMeta || tok.typ == itemRightMeta {
		return nil, nil
	}
	if tok.typ != itemIdentifier {
		return nil, fmt.Errorf("found %q [%v], expected identifier", lit, tok.typ)
	}
	stmt.key = lit

	tok, lit = p.getTokLit()
	if tok.typ != itemEquals {
		return nil, fmt.Errorf("found %q, expected equals", lit)
	}
	tok, lit = p.getTokLit()
	if tok.typ != itemBool && tok.typ != itemNumber && tok.typ != itemString {
		return nil, fmt.Errorf("found %q [%v], expected value(bool, number, string)", lit, tok.typ)
	}
	stmt.value = lit
	stmt.typ = tok.typ

	return stmt, nil

}

type numberType int

// Types of numbers that could be parsed
const (
	INT numberType = iota
	FLOAT
	ERR
)

//parseNumber parses a *selectStatement and returns the number type as a numberType object
//This is really ghetto
func parseNumber(stmt *selectStatement) numberType {
	_, err := strconv.ParseInt(stmt.value, 0, 64)
	if err == nil {
		return INT
	}
	fmt.Println(err)
	_, err = strconv.ParseFloat(stmt.value, 64)
	if err == nil {
		return FLOAT
	}
	fmt.Println(err)
	return ERR
}
