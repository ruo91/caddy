package parse

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// Dispenser is a type that dispenses tokens, similarly to a lexer,
// except that it can do so with some notion of structure and has
// some really convenient methods.
type Dispenser struct {
	filename string
	tokens   []token
	cursor   int
	nesting  int
}

// NewDispenser returns a Dispenser, ready to use for parsing the given input.
func NewDispenser(filename string, input io.Reader) Dispenser {
	return Dispenser{
		filename: filename,
		tokens:   allTokens(input),
		cursor:   -1,
	}
}

// NewDispenserTokens returns a Dispenser filled with the given tokens.
func NewDispenserTokens(filename string, tokens []token) Dispenser {
	return Dispenser{
		filename: filename,
		tokens:   tokens,
		cursor:   -1,
	}
}

// Next loads the next token. Returns true if a token
// was loaded; false otherwise. If false, all tokens
// have already been consumed.
func (d *Dispenser) Next() bool {
	if d.cursor < len(d.tokens)-1 {
		d.cursor++
		return true
	}
	return false
}

// NextArg loads the next token if it is on the same
// line. Returns true if a token was loaded; false
// otherwise. If false, all tokens on the line have
// been consumed.
func (d *Dispenser) NextArg() bool {
	if d.cursor < 0 {
		d.cursor++
		return true
	}
	if d.cursor >= len(d.tokens) {
		return false
	}
	if d.cursor < len(d.tokens)-1 &&
		(d.tokens[d.cursor].line+d.numLineBreaks(d.cursor) == d.tokens[d.cursor+1].line) {
		d.cursor++
		return true
	}
	return false
}

// NextLine loads the next token only if it is not on the same
// line as the current token, and returns true if a token was
// loaded; false otherwise. If false, there is not another token
// or it is on the same line.
func (d *Dispenser) NextLine() bool {
	if d.cursor < 0 {
		d.cursor++
		return true
	}
	if d.cursor >= len(d.tokens) {
		return false
	}
	if d.cursor < len(d.tokens)-1 &&
		d.tokens[d.cursor].line+d.numLineBreaks(d.cursor) < d.tokens[d.cursor+1].line {
		d.cursor++
		return true
	}
	return false
}

// NextBlock can be used as the condition of a for loop
// to load the next token as long as it opens a block or
// is already in a block. It returns true if a token was
// loaded, or false when the block's closing curly brace
// was loaded and thus the block ended. Nested blocks are
// not supported.
func (d *Dispenser) NextBlock() bool {
	if d.nesting > 0 {
		d.Next()
		if d.Val() == "}" {
			d.nesting--
			return false
		}
		return true
	}
	if !d.NextArg() { // block must open on same line
		return false
	}
	if d.Val() != "{" {
		d.cursor-- // roll back if not opening brace
		return false
	}
	d.Next()
	d.nesting++
	return true
}

// Val gets the text of the current token. If there is no token
// loaded, it returns empty string.
func (d *Dispenser) Val() string {
	if d.cursor < 0 || d.cursor >= len(d.tokens) {
		return ""
	}
	return d.tokens[d.cursor].text
}

// Line gets the line number of the current token. If there is no token
// loaded, it returns 0.
func (d *Dispenser) Line() int {
	if d.cursor < 0 || d.cursor >= len(d.tokens) {
		return 0
	}
	return d.tokens[d.cursor].line
}

// Args is a convenience function that loads the next arguments
// (tokens on the same line) into an arbitrary number of strings
// pointed to in targets. If there are fewer tokens available
// than string pointers, the remaining strings will not be changed
// and false will be returned. If there were enough tokens available
// to fill the arguments, then true will be returned.
func (d *Dispenser) Args(targets ...*string) bool {
	enough := true
	for i := 0; i < len(targets); i++ {
		if !d.NextArg() {
			enough = false
			break
		}
		*targets[i] = d.Val()
	}
	return enough
}

// RemainingArgs loads any more arguments (tokens on the same line)
// into a slice and returns them. Open curly brace tokens also indicate
// the end of arguments, and the curly brace is not included in
// the return value nor is it loaded.
func (d *Dispenser) RemainingArgs() []string {
	var args []string

	for d.NextArg() {
		if d.Val() == "{" {
			d.cursor--
			break
		}
		args = append(args, d.Val())
	}

	return args
}

// ArgErr returns an argument error, meaning that another
// argument was expected but not found. In other words,
// a line break or open curly brace was encountered instead of
// an argument.
func (d *Dispenser) ArgErr() error {
	if d.Val() == "{" {
		return d.Err("Unexpected token '{', expecting argument")
	}
	return d.Errf("Wrong argument count or unexpected line ending after '%s'", d.Val())
}

// SyntaxErr creates a generic syntax error which explains what was
// found and what was expected.
func (d *Dispenser) SyntaxErr(expected string) error {
	msg := fmt.Sprintf("%s:%d - Syntax error: Unexpected token '%s', expecting '%s'", d.filename, d.Line(), d.Val(), expected)
	return errors.New(msg)
}

// EofErr returns an EOF error, meaning that end of input
// was found when another token was expected.
func (d *Dispenser) EofErr() error {
	return d.Errf("Unexpected EOF")
}

// Err generates a custom parse error with a message of msg.
func (d *Dispenser) Err(msg string) error {
	msg = fmt.Sprintf("%s:%d - Parse error: %s", d.filename, d.Line(), msg)
	return errors.New(msg)
}

// Errf is like Err, but for formatted error messages
func (d *Dispenser) Errf(format string, args ...interface{}) error {
	return d.Err(fmt.Sprintf(format, args...))
}

// numLineBreaks counts how many line breaks are in the token
// value given by the token index tknIdx. It returns 0 if the
// token does not exist or there are no line breaks.
func (d *Dispenser) numLineBreaks(tknIdx int) int {
	if tknIdx < 0 || tknIdx >= len(d.tokens) {
		return 0
	}
	return strings.Count(d.tokens[tknIdx].text, "\n")
}
