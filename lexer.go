package ariaconfig

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type item struct {
	typ itemType
	val string
}

func (i item) String() string {
	switch {
	case i.typ == itemEOF:
		return "EOF"
	case i.typ == itemError:
		return i.val
	case len(i.val) > 10:
		return fmt.Sprintf("%.10q...", i.val)
	}

	return fmt.Sprintf("%q", i.val)
}

type lexer struct {
	name  string //used for error reports
	input string // input string
	start int    // start position
	pos   int    //current position
	width int    // width
	state stateFn
	items chan item // channel of scanned items
}

type itemType int

//entire file is k = v;
const (
	itemError      itemType = iota //error occured; value is text of error
	itemString                     //quoted string (includes quotes)
	itemNumber                     //number constant
	itemEquals                     //denoting left is a key and right is a value
	itemText                       //useless string
	itemLeftMeta                   //start of what we need
	itemRightMeta                  //end of what we need
	itemBool                       //True or false
	itemIdentifier                 //name of something
	itemEOF
)

const eof = -1

const (
	leftMeta  = "{{"
	rightMeta = "}}"
)

type stateFn func(*lexer) stateFn

func lex(name, input string) *lexer {
	l := &lexer{
		name:  name,
		input: input,
		state: lexText,
		items: make(chan item, 2),
	}
	//strip newlines
	l.input = strings.Replace(l.input, "\n", " ", -1)
	go l.run() // Concurrently run state machine
	return l
}

func (l *lexer) run() {
	for state := lexText; state != nil; {
		state = state(l)
	}

	close(l.items) // No more tokens coming
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], leftMeta) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexLeftMeta // Next state
		}

		if l.next() == eof {
			break
		}
	}

	//Correctly reached EOF
	if l.pos > l.start {
		l.emit(itemText)
	}

	l.emit(itemEOF)
	return nil
}

func lexLeftMeta(l *lexer) stateFn {
	l.pos += len(leftMeta)
	l.emit(itemLeftMeta)
	return lexInsideAction
}

func lexRightMeta(l *lexer) stateFn {
	l.pos += len(rightMeta)
	l.emit(itemRightMeta)
	return lexText
}

func lexInsideAction(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.input[l.pos:], rightMeta) {
			return lexRightMeta
		}

		switch r := l.next(); {
		case r == eof:
			return l.errorf("unclosed action @ %v [%v]", l.pos, string(r))
		case isSpace(r) || r == '\n':
			l.ignore()
		case r == '=':
			return lexEquals
		case r == '"':
			return lexQuote
		case r == '+' || r == '-' || '0' <= r && r <= '9':
			l.backup()
			return lexNumber
		case isAlphaNumeric(r):
			l.backup()
			return lexIdentifier
		}
	}
}
func lexEquals(l *lexer) stateFn {
	l.pos += len("=")
	l.emit(itemEquals)
	return lexInsideAction
}
func lexQuote(l *lexer) stateFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated quoted string")
		case '"':
			break Loop
		}
	}

	l.emit(itemString)
	return lexInsideAction
}

func lexIdentifier(l *lexer) stateFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			//absorb
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			switch {
			case word == "true", word == "false":
				l.emit(itemBool)
			default:
				l.emit(itemIdentifier)
			}
			break Loop
		}
	}

	return lexInsideAction
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width
	return r
}

func (l *lexer) ignore() {
	l.start = l.pos
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

//accept consumed the next rune
//if its from the valid set
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func lexNumber(l *lexer) stateFn {
	//Optional leading sign
	l.accept("+-")
	//Is it hex?
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}

	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}

	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}

	l.accept("i")

	if isAlphaNumeric(l.peek()) {
		l.next()
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	}

	l.emit(itemNumber)
	return lexInsideAction
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}

	return nil
}

func (l *lexer) nextItem() item {
	for {
		select {
		case item := <-l.items:
			return item
		default:
			//fmt.Println("at shit", l)
			if l.state == nil {
				fmt.Println("state is nil!")
			}
			l.state = l.state(l)
		}
	}
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
